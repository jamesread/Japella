package japella;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;
import java.lang.reflect.Method;
import java.util.ArrayList;
import java.util.Timer;
import java.util.TimerTask;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public abstract class MessagePlugin {
	@Retention(RetentionPolicy.RUNTIME)
	@Target(ElementType.METHOD)
	public static @interface CommandMessage {
		String keyword() default "";

		MessageTarget target() default MessageTarget.ANYWHERE;
	}

	public static class MessagePluginTimer extends TimerTask {
		private final Bot bot;
		private final String channel;
		private final MessagePlugin plugin;
		private final Timer timer;

		public MessagePluginTimer(Bot bot, String channel, MessagePlugin plugin, int period) {
			this.bot = bot;
			this.channel = channel;
			this.plugin = plugin;

			this.timer = new Timer(bot.getName() + " MP timer for " + plugin.getClass().getSimpleName() + " with period of " + period, true);

			this.timer.scheduleAtFixedRate(this, 30, period);
		}

		public void onTimerTick() {
			this.plugin.onTimerTick(this.bot, this.channel);
		}

		@Override
		public void run() {
			this.onTimerTick();
		}
	}

	public static enum MessageTarget {
		PM, CHAT, ANYWHERE;
	}

	private static final transient Logger LOG = LoggerFactory.getLogger(MessagePlugin.class);

	public abstract void addMessage(String m);

	void callCommandMessages(String channel, String sender, String input) {
		Command command = new Command(input);

		for (Method m : this.getMethods()) {
			String keyword = m.getAnnotation(CommandMessage.class).keyword();

			if (keyword.isEmpty()) {
				keyword = m.getName();
			}

			if (command.isKeyword(keyword)) {
				if ((m.getGenericParameterTypes().length < 1) || (m.getGenericParameterTypes()[0] != String.class)) {
					MessagePlugin.LOG.debug("InputSelector method found for " + input + ", but this method needs to take a string argument");
				} else {
					try {
						m.invoke(this, channel, sender, command);
						return;
					} catch (Exception e) {
						e.printStackTrace();
					}
				}
			}
		}

		System.out.println("Could not find: " + input);
	}

	private ArrayList<Method> getMethods() {
		ArrayList<Method> methods = new ArrayList<>();

		for (Method m : this.getClass().getMethods()) {
			if (m.isAnnotationPresent(CommandMessage.class)) {
				methods.add(m);
			}
		}

		return methods;
	}

	public String getName() {
		return this.getClass().getSimpleName();
	}

	public abstract void onMessage(Bot bot, String channel, String sender, String login, String hostname, String message);

	public abstract void onPrivateMessage(Bot bot, String sender, String message);

	public abstract void onTimerTick(Bot bot, String channel);
}
