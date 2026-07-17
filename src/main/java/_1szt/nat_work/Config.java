package _1szt.nat_work;

import java.util.List;
// import java.util.Set;
// import java.util.stream.Collectors;

import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.resources.ResourceLocation;
// import net.minecraft.world.item.Item;
// import net.neoforged.bus.api.SubscribeEvent;
// import net.neoforged.fml.common.EventBusSubscriber;
// import net.neoforged.fml.event.config.ModConfigEvent;
import net.neoforged.neoforge.common.ModConfigSpec;

/**
 * 模组配置文件类。
 * 使用 NeoForge 的 ModConfigSpec API 定义配置项，
 * 所有配置将自动生成在 config/nat_work-common.toml 中。
 */
public class Config {
    private static final ModConfigSpec.Builder BUILDER = new ModConfigSpec.Builder();

    /** 是否在通用初始化时输出泥土方块的注册信息 */
    public static final ModConfigSpec.BooleanValue LOG_DIRT_BLOCK = BUILDER
            .comment("是否在通用初始化时输出泥土方块的注册信息")
            .define("logDirtBlock", true);

    /** 魔法数字，一个整型配置值，范围 0 ~ 2^31-1 */
    public static final ModConfigSpec.IntValue MAGIC_NUMBER = BUILDER
            .comment("一个魔法数字（整型）")
            .defineInRange("magicNumber", 42, 0, Integer.MAX_VALUE);

    /** 魔法数字的前导介绍文本 */
    public static final ModConfigSpec.ConfigValue<String> MAGIC_NUMBER_INTRODUCTION = BUILDER
            .comment("魔法数字的介绍文本")
            .define("magicNumberIntroduction", "魔法数字是... ");

    /** 物品 ID 字符串列表，将在通用初始化时依次输出 */
    public static final ModConfigSpec.ConfigValue<List<? extends String>> ITEM_STRINGS = BUILDER
            .comment("需要在通用初始化时输出的物品列表")
            .defineListAllowEmpty("items", List.of("minecraft:iron_ingot"), () -> "", Config::validateItemName);

    /** 构建完成的配置规范实例 */
    static final ModConfigSpec SPEC = BUILDER.build();

    /**
     * 验证物品名称是否有效（必须在注册表中存在）。
     *
     * @param obj 待验证的对象
     * @return 如果是有效的物品 ID 则返回 true
     */
    private static boolean validateItemName(final Object obj) {
        return obj instanceof String itemName && BuiltInRegistries.ITEM.containsKey(ResourceLocation.parse(itemName));
    }
}
