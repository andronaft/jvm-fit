package demo;

import java.lang.management.BufferPoolMXBean;
import java.lang.management.ManagementFactory;
import java.lang.management.MemoryUsage;
import java.nio.ByteBuffer;
import java.util.ArrayList;
import java.util.List;

import org.springframework.boot.ApplicationArguments;
import org.springframework.boot.ApplicationRunner;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;

/**
 * A small Spring Boot service with the same memory load under any JVM settings:
 * it keeps LOAD_HEAP_MB in a heap cache and LOAD_DIRECT_MB in direct buffers
 * (what Netty and NIO clients do), then logs its memory use every second.
 */
@SpringBootApplication
public class OrdersApplication {
    private static final int MB = 1024 * 1024;
    // Small blocks, like cache entries. G1 stores arrays of half a region or
    // more as humongous objects, which would waste part of a small heap.
    private static final int BLOCK = 64 * 1024;
    private static final int STEPS = 20;

    private final List<byte[]> heapCache = new ArrayList<>();
    private final List<ByteBuffer> directCache = new ArrayList<>();

    public static void main(String[] args) {
        SpringApplication.run(OrdersApplication.class, args);
    }

    @Bean
    ApplicationRunner load() {
        return (ApplicationArguments args) -> {
            int heapMb = envInt("LOAD_HEAP_MB", 260);
            int directMb = envInt("LOAD_DIRECT_MB", 140);
            Thread t = new Thread(() -> run(heapMb, directMb), "load");
            t.setDaemon(true);
            t.start();
        };
    }

    private void run(int heapMb, int directMb) {
        try {
            for (int i = 1; i <= STEPS; i++) {
                addHeap(heapMb / STEPS);
                addDirect(directMb / STEPS);
                status();
                Thread.sleep(1000);
            }
            System.out.printf("load complete: %dMi in the heap, %dMi in direct buffers%n", heapMb, directMb);
            while (true) {
                status();
                Thread.sleep(5000);
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        } catch (OutOfMemoryError e) {
            // Never look healthy without the load in place.
            System.out.println("load failed: " + e);
            Runtime.getRuntime().halt(1);
        }
    }

    private void addHeap(int mb) {
        for (int i = 0; i < mb * MB / BLOCK; i++) {
            byte[] block = new byte[BLOCK];
            for (int p = 0; p < BLOCK; p += 4096) {
                block[p] = 1;
            }
            heapCache.add(block);
        }
    }

    private void addDirect(int mb) {
        for (int i = 0; i < mb * MB / BLOCK; i++) {
            ByteBuffer block = ByteBuffer.allocateDirect(BLOCK);
            for (int p = 0; p < BLOCK; p += 4096) {
                block.put(p, (byte) 1);
            }
            directCache.add(block);
        }
    }

    private void status() {
        System.gc();
        MemoryUsage heap = ManagementFactory.getMemoryMXBean().getHeapMemoryUsage();
        MemoryUsage nonHeap = ManagementFactory.getMemoryMXBean().getNonHeapMemoryUsage();
        long direct = ManagementFactory.getPlatformMXBeans(BufferPoolMXBean.class).stream()
                .filter(b -> b.getName().equals("direct"))
                .mapToLong(BufferPoolMXBean::getMemoryUsed).sum();
        System.out.printf("heap %dMi of %dMi (%d%%), direct buffers %dMi, metaspace and code %dMi%n",
                heap.getUsed() / MB, heap.getMax() / MB, heap.getUsed() * 100 / heap.getMax(),
                direct / MB, nonHeap.getUsed() / MB);
    }

    private static int envInt(String name, int fallback) {
        String v = System.getenv(name);
        return v == null ? fallback : Integer.parseInt(v);
    }
}
