package _1szt.nat_work;

import java.io.File;
import java.io.IOException;

import org.slf4j.Logger;

import com.mojang.logging.LogUtils;

public class NetWorkNativeLauncher {
    private static final Logger LOGGER = LogUtils.getLogger();

    public static void launch() {
        try {
            String osName = System.getProperty("os.name").toLowerCase();
            String osArch = System.getProperty("os.arch").toLowerCase();

            String binaryName = getBinaryName(osName, osArch);
            if (binaryName == null) {
                LOGGER.error("不支持的平台: {} {}", osName, osArch);
                return;
            }

            String binaryPath = "net_work" + File.separator + binaryName;
            File binaryFile = new File(binaryPath);

            if (!binaryFile.exists()) {
                LOGGER.error("未找到本地二进制文件: {}", binaryFile.getAbsolutePath());
                return;
            }

            // 确保可执行权限（Unix 系统）
            if (!binaryFile.canExecute()) {
                binaryFile.setExecutable(true);
            }

            // 构建启动命令
            ProcessBuilder pb = new ProcessBuilder(
                    binaryFile.getAbsolutePath(),
                    "-mode", "client",
                    "-port", "127.0.0.1:25565",
                    "-addr", "cn-0.dns.1szt.com:24688"
            );

            // 继承 Minecraft 进程的 IO
            pb.inheritIO();

            // 启动进程
            Process process = pb.start();

            LOGGER.info("已启动本地二进制文件: {} (PID: {})", binaryName, process.pid());

            // 注册关闭钩子，在 JVM 退出时终止子进程
            Runtime.getRuntime().addShutdownHook(new Thread(() -> {
                if (process.isAlive()) {
                    process.destroy();
                    LOGGER.info("已停止本地二进制文件: {}", binaryName);
                }
            }));

        } catch (IOException e) {
            LOGGER.error("启动本地二进制文件失败", e);
        }
    }

    /**
     * 根据操作系统和架构返回对应的二进制文件名
     */
    private static String getBinaryName(String osName, String osArch) {
        boolean isArm64 = osArch.contains("arm64") || osArch.contains("aarch64");
        boolean isAmd64 = osArch.contains("amd64") || osArch.contains("x86_64")
                || (osArch.contains("x86") && !isArm64);

        // Windows (仅 amd64)
        if (osName.contains("win") && isAmd64) {
            return "net_work-windows-amd64.exe";
        }

        // macOS (内核名 darwin) - amd64 和 arm64 都有
        if (osName.contains("mac")) {
            return isArm64 ? "net_work-darwin-arm64" : "net_work-darwin-amd64";
        }

        // Android (仅 arm64)
        if (osName.contains("android")) {
            return "net_work-android-arm64";
        }

        // Linux (仅 amd64)
        if ((osName.contains("nix") || osName.contains("nux") || osName.contains("aix")) && isAmd64) {
            return "net_work-linux-amd64";
        }

        return null;
    }
}
