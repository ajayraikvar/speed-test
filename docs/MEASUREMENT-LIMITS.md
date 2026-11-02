# Measurement limits and interpretation

1. Throughput is computed from bytes delivered/accepted by the browser over elapsed time. It includes browser scheduling, HTTP, TCP, server, and operating-system overhead.
2. Multiple streams can raise throughput by using parallel TCP connections, but can also increase CPU/memory usage and should be compared using consistent stream counts.
3. Bidirectional mode creates competing upload and download traffic simultaneously. Each direction is measured over its own active interval.
4. Latency is the median/minimum/maximum of repeated HTTP round trips to the host. Jitter is the average absolute difference between consecutive HTTP round-trip measurements. These are not ICMP ping or standardized RTP jitter measurements.
5. The browser network API is optional and inconsistently supported. It may expose a coarse effective network type rather than physical Ethernet/Wi-Fi.
6. No raw UDP packet loss, PHY link speed, Wi-Fi signal strength, duplex negotiation, or hardware interface counters are measured.
7. For reliable comparisons, run both PCs on the same LAN, preferably wired, close background transfers, keep duration/stream settings the same, and run more than once.
