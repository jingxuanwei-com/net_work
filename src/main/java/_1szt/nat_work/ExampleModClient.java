package _1szt.nat_work;

import net.minecraft.client.Minecraft;
import net.neoforged.api.distmarker.Dist;
import net.neoforged.bus.api.SubscribeEvent;
import net.neoforged.fml.ModContainer;
import net.neoforged.fml.common.EventBusSubscriber;
import net.neoforged.fml.common.Mod;
import net.neoforged.fml.event.lifecycle.FMLClientSetupEvent;
import net.neoforged.neoforge.client.gui.ConfigurationScreen;
import net.neoforged.neoforge.client.gui.IConfigScreenFactory;

/**
 * 客户端专属模组初始化类。
 * 此类不会在专用服务器上加载，因此可以安全地访问客户端代码。
 */
@Mod(value = ExampleMod.MODID, dist = Dist.CLIENT)
// 使用 @EventBusSubscriber 自动注册本类中所有带 @SubscribeEvent 注解的静态方法
@EventBusSubscriber(modid = ExampleMod.MODID, value = Dist.CLIENT)
public class ExampleModClient {
    /**
     * 客户端模组构造方法。
     * 注册配置界面工厂，使玩家能在 Mod 列表中点击"配置"按钮打开本模组的配置界面。
     *
     * @param container 模组容器，用于注册扩展点
     */
    public ExampleModClient(ModContainer container) {
        // 注册配置界面工厂，NeoForge 将自动创建配置界面
        // 在 Mod 列表 → 点击本模组 → 点击"配置"按钮即可打开
        // 注意：别忘了在 en_us.json 中添加配置项的翻译文本
        container.registerExtensionPoint(IConfigScreenFactory.class, ConfigurationScreen::new);
    }

    /**
     * 客户端初始化事件处理器。
     * 在游戏客户端启动时调用，适合放置渲染注册、按键绑定等客户端逻辑。
     */
    @SubscribeEvent
    static void onClientSetup(FMLClientSetupEvent event) {
        ExampleMod.LOGGER.info("=== 客户端初始化完成 ===");
        ExampleMod.LOGGER.info("当前玩家 >> {}", Minecraft.getInstance().getUser().getName());
    }
}
