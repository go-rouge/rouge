// a single-line comment
/* a multiline
   comment */
package com.example.app;

import java.util.List;
import java.util.*;

@Deprecated
public class Widget extends Base implements Runnable {
    private static final int MAX_SIZE = 10;
    protected volatile boolean running = false;
    char initial = 'x';
    char escaped = '\n';
    String name = "a \"quoted\" value";
    double ratio = 1_000.5e3d;
    float small = 0.5f;
    int hex = 0xFF_FF;
    int bin = 0b1010_1010;
    int oct = 0777;
    long big = 1_234L;
    var inferred = List.of(1, 2);

    public static void main(String[] args) {
        Widget w = new Widget();
        outer:
        for (int i = 0; i < MAX_SIZE; i++) {
            if (i % 2 == 0) continue outer;
            switch (i) {
                case 1: break;
                default: throw new IllegalStateException("bad");
            }
        }
        try {
            w.run();
        } catch (Exception e) {
            assert e != null;
        } finally {
            w.close();
        }
    }

    @Override
    public void run() {
        String text = """
            a text block
            with "quotes" inside
            """;
        System.out.println(text);
    }

    private synchronized void close() throws Exception {
        this.running = false;
    }
}

interface Shape {
    double area();
}

record Point(int x, int y) {}

enum Colour { RED, GREEN, BLUE }
