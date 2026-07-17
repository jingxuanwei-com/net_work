package _1szt.motd;

import java.io.File;

public class Motd {
    public static void Run() {
        System.out.println("=== 1szt ===");

        System.out.println(" __  _   ___   _____     _   _    __    ___   _  __ ");
        System.out.println("|  \\| | | __| |_   _|   | | | |  /__\\  | _ \\ | |/ / ");
        System.out.println("| | ' | | _|    | |     | 'V' | | \\/ | | v / |   <  ");
        System.out.println("|_|\\__| |___|   |_|     !_/ \\_!  \\__/  |_|_\\ |_|\\_\\ ");

        System.out.println(">>> NET WORK <<<");

        System.out.println("Current path: " + new File(".").getAbsolutePath());
    }
}
