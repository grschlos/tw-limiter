//go:build ignore

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/in.h>

// Structure to hold dynamic configuration passed from Go
struct config_val {
    __u64 max_packets;
    __u64 interval_ns;
};

// Control Map to store the configuration (single entry at index 0)
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, struct config_val);
    __uint(max_entries, 1);
} config_map SEC(".maps");

// Map to keep track of packet counts per IP address
struct ip_counter {
    __u64 last_reset;
    __u64 count;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);
    __type(value, struct ip_counter);
    __uint(max_entries, 10000);
} ip_limits SEC(".maps");

SEC("xdp")
int count_packets(struct xdp_md *ctx) {
    // 1. Read dynamic configuration from the control map
    __u32 config_idx = 0;
    struct config_val *cfg = bpf_map_lookup_elem(&config_map, &config_idx);

    // Apply failsafe default values in case Go hasn't written the config yet
    __u64 max_p = 100;
    __u64 int_ns = 1000000000; // 1 second in nanoseconds

    if (cfg) {
        max_p = cfg->max_packets;
        int_ns = cfg->interval_ns;
    }

    // 2. Parse the network packet (Ethernet -> IPv4)
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;

    struct iphdr *iph = data + sizeof(struct ethhdr);
    if ((void *)(iph + 1) > data_end)
        return XDP_PASS;

    __u32 ip = iph->saddr;
    __u64 now = bpf_ktime_get_ns();

    // 3. Look up the current IP in the hash map
    struct ip_counter *counter = bpf_map_lookup_elem(&ip_limits, &ip);
    if (!counter) {
        // First time seeing this IP: initialize counter
        struct ip_counter new_counter = {
            .last_reset = now,
            .count = 1,
        };
        bpf_map_update_elem(&ip_limits, &ip, &new_counter, BPF_ANY);
        return XDP_PASS;
    }

    // 4. Reset counter if the time interval has elapsed
    if (now - counter->last_reset > int_ns) {
        counter->last_reset = now;
        counter->count = 1;
        return XDP_PASS;
    }

    // 5. Increment and enforce the dynamic limit
    counter->count++;
    if (counter->count > max_p) {
        // The packet threshold is exceeded. Drop the packet.
        return XDP_DROP;
    }

    return XDP_PASS;
}

char __license[] SEC("license") = "Dual MIT/GPL";
