//go:build ignore

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <bpf/bpf_helpers.h>

struct counter {
    __u64 last_reset;
    __u64 count;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10240);
    __type(key, __u32);
    __type(value, struct counter);
} ip_limits SEC(".maps");

// Thresholds (could be updated from Go via another map later)
const __u64 interval_ns = 1000000000; // 1 second
const __u64 max_packets = 100;

SEC("xdp")
int count_packets(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data     = (void *)(long)ctx->data;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end) return XDP_PASS;
    if (eth->h_proto != 0x0008) return XDP_PASS; // Using constant for ETH_P_IP (0x0800) in Big Endian

    struct iphdr *iph = (void *)(eth + 1);
    if ((void *)(iph + 1) > data_end) return XDP_PASS;

    __u32 ip = iph->saddr;
    __u64 now = bpf_ktime_get_ns();

    struct counter *c = bpf_map_lookup_elem(&ip_limits, &ip);
    if (c) {
        // Check if the time window has passed
        if (now - c->last_reset > interval_ns) {
            c->last_reset = now;
            c->count = 1;
            return XDP_PASS;
        }

        if (c->count >= max_packets) {
            return XDP_DROP;
        }
        
        __sync_fetch_and_add(&c->count, 1);
    } else {
        struct counter new_c = { .last_reset = now, .count = 1 };
        bpf_map_update_elem(&ip_limits, &ip, &new_c, BPF_ANY);
    }

    return XDP_PASS;
}

char __license[] SEC("license") = "Dual MIT/GPL";
