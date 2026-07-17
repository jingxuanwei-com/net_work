package _1szt.nat_work;

import org.slf4j.Logger;

import com.mojang.logging.LogUtils;

import _1szt.motd.Motd;

import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.core.registries.Registries;
import net.minecraft.network.chat.Component;
import net.minecraft.world.food.FoodProperties;
import net.minecraft.world.item.BlockItem;
import net.minecraft.world.item.CreativeModeTab;
import net.minecraft.world.item.CreativeModeTabs;
import net.minecraft.world.item.Item;
import net.minecraft.world.level.block.Block;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockBehaviour;
import net.minecraft.world.level.material.MapColor;
// import net.neoforged.api.distmarker.Dist;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.bus.api.SubscribeEvent;
import net.neoforged.fml.common.Mod;
import net.neoforged.fml.config.ModConfig;
import net.neoforged.fml.ModContainer;
import net.neoforged.fml.event.lifecycle.FMLCommonSetupEvent;
import net.neoforged.neoforge.common.NeoForge;
import net.neoforged.neoforge.event.BuildCreativeModeTabContentsEvent;
import net.neoforged.neoforge.event.server.ServerStartingEvent;
import net.neoforged.neoforge.registries.DeferredBlock;
import net.neoforged.neoforge.registries.DeferredHolder;
import net.neoforged.neoforge.registries.DeferredItem;
import net.neoforged.neoforge.registries.DeferredRegister;

// 此值必须与 META-INF/neoforge.mods.toml 文件中的条目一致
@Mod(ExampleMod.MODID)
public class ExampleMod {
    // 定义模组 ID，所有地方通过此常量引用
    public static final String MODID = "nat_work";
    // 直接引用 SLF4J 日志记录器，用于输出日志信息
    public static final Logger LOGGER = LogUtils.getLogger();
    // 创建延迟注册表，用于注册方块（Blocks），所有方块将在 "nat_work" 命名空间下注册
    public static final DeferredRegister.Blocks BLOCKS = DeferredRegister.createBlocks(MODID);
    // 创建延迟注册表，用于注册物品（Items），所有物品将在 "nat_work" 命名空间下注册
    public static final DeferredRegister.Items ITEMS = DeferredRegister.createItems(MODID);
    // 创建延迟注册表，用于注册创造模式标签页（CreativeModeTabs）
    public static final DeferredRegister<CreativeModeTab> CREATIVE_MODE_TABS = DeferredRegister.create(Registries.CREATIVE_MODE_TAB, MODID);

    // 注册一个ID为 "nat_work:example_block" 的简单方块，材质为石头颜色
    public static final DeferredBlock<Block> EXAMPLE_BLOCK = BLOCKS.registerSimpleBlock("example_block", BlockBehaviour.Properties.of().mapColor(MapColor.STONE));
    // 为上述方块注册对应的方块物品（BlockItem），ID 同样为 "nat_work:example_block"
    public static final DeferredItem<BlockItem> EXAMPLE_BLOCK_ITEM = ITEMS.registerSimpleBlockItem("example_block", EXAMPLE_BLOCK);

    // 注册一个ID为 "nat_work:example_item" 的食物物品，营养值1，饱和度2，且始终可食用
    public static final DeferredItem<Item> EXAMPLE_ITEM = ITEMS.registerSimpleItem("example_item", new Item.Properties().food(new FoodProperties.Builder()
            .alwaysEdible().nutrition(1).saturationModifier(2f).build()));

    // 注册一个ID为 "nat_work:example_tab" 的创造模式标签页，放置于战斗标签页之后
    public static final DeferredHolder<CreativeModeTab, CreativeModeTab> EXAMPLE_TAB = CREATIVE_MODE_TABS.register("example_tab", () -> CreativeModeTab.builder()
            .title(Component.translatable("itemGroup.nat_work")) // 标签页标题的语言键
            .withTabsBefore(CreativeModeTabs.COMBAT)             // 放在战斗标签页之前
            .icon(() -> EXAMPLE_ITEM.get().getDefaultInstance()) // 标签页图标
            .displayItems((parameters, output) -> {
                output.accept(EXAMPLE_ITEM.get()); // 将示例物品添加到该标签页中
            }).build());

    /**
     * 模组类的构造函数，在模组加载时第一个执行。
     * FML 会自动识别 IEventBus 和 ModContainer 等参数类型并传入。
     *
     * @param modEventBus  模组事件总线，用于监听模组生命周期事件
     * @param modContainer 模组容器，用于注册配置等
     */
    public ExampleMod(IEventBus modEventBus, ModContainer modContainer) {
        // 注册 commonSetup 方法，在模组通用初始化阶段调用
        modEventBus.addListener(this::commonSetup);

        // 将所有延迟注册表注册到模组事件总线，使方块、物品、标签页能够按顺序自动注册
        BLOCKS.register(modEventBus);
        ITEMS.register(modEventBus);
        CREATIVE_MODE_TABS.register(modEventBus);

        // 在 NeoForge 全局事件总线上注册本类，以接收服务器启动等全局事件
        // 注意：只有当本类中有 @SubscribeEvent 注解的方法时才需要此行
        NeoForge.EVENT_BUS.register(this);

        // 注册物品到创造模式标签页的事件监听
        modEventBus.addListener(this::addCreative);

        // 注册模组的通用配置（ModConfigSpec），FML 将据此创建和加载配置文件
        modContainer.registerConfig(ModConfig.Type.COMMON, Config.SPEC);
    }

    /**
     * 通用初始化方法，在模组加载的通用阶段调用。
     * 适合放置与客户端/服务端无关的初始化逻辑。
     */
    private void commonSetup(FMLCommonSetupEvent event) {
        // 打印 MOTD 图案
        Motd.Run();

        LOGGER.info("=== 通用初始化完成 ===");

        // 根据配置决定是否记录泥土方块的注册信息
        if (Config.LOG_DIRT_BLOCK.getAsBoolean()) {
            LOGGER.info("泥土方块注册键 >> {}", BuiltInRegistries.BLOCK.getKey(Blocks.DIRT));
        }

        // 输出配置中的魔法数字
        LOGGER.info("{}{}", Config.MAGIC_NUMBER_INTRODUCTION.get(), Config.MAGIC_NUMBER.getAsInt());

        // 遍历并输出配置中的所有物品字符串
        Config.ITEM_STRINGS.get().forEach((item) -> LOGGER.info("配置物品 >> {}", item));
    }

    /**
     * 将示例方块物品添加到"建筑方块"创造模式标签页中。
     */
    private void addCreative(BuildCreativeModeTabContentsEvent event) {
        if (event.getTabKey() == CreativeModeTabs.BUILDING_BLOCKS) {
            event.accept(EXAMPLE_BLOCK_ITEM);
        }
    }

    /**
     * 服务端启动事件处理器。
     * 当游戏服务端启动时会自动调用此方法。
     */
    @SubscribeEvent
    public void onServerStarting(ServerStartingEvent event) {
        LOGGER.info("服务端正在启动...");
    }
}
