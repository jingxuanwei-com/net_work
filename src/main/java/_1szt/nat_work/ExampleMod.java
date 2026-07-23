package _1szt.nat_work;

import org.slf4j.Logger;

import com.mojang.logging.LogUtils;

import _1szt.motd.Motd;

import net.neoforged.fml.common.Mod;

// 此值必须与 META-INF/neoforge.mods.toml 文件中的条目一致
@Mod(ExampleMod.MODID)
public class ExampleMod {
    // 定义字符串 MODID，所有地方通过此常量引用
    public static final String MODID = "nat_work";

    // 定义 SLF4J 日志记录器，用于输出日志信息
    public static final Logger LOGGER = LogUtils.getLogger();

    // 构造函数，Mod 加载时调用
    public ExampleMod() {
        Motd.Run();
    }

}
