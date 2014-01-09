package japella;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;
import java.lang.reflect.Method;
import java.lang.reflect.Type;
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

	public static class Message {
		public final Bot bot;
		public final String channel;
		public final String sender;
		public final Command command;

		public Message(Bot bot, String channel2, String sender2, Command command2) {
			this.bot = bot;
			this.channel = channel2;
			this.sender = sender2;
			this.command = command2;
		}

		public void reply(String message) {
			this.bot.sendMessageResponsibly(this.channel, message);
		}
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

	public void callCommandMessages(Bot bot, String channel, String sender, String input) {
		Command command = new Command(input);

		for (Method method : this.getCommandMessageMethod(command)) {
			Type[] types = method.getGenericParameterTypes();

			if ((types.length != 1) && (types[0] != Message.class)) {
				MessagePlugin.LOG.debug("InputSelector method found for " + input + ", but this method needs to take a single Message argument");
			} else {
				try {
					method.invoke(this, new Message(bot, channel, sender, command));
					return;
				} catch (Exception e) {
					e.printStackTrace();
				}
			}
		}

		MessagePlugin.LOG.warn("Could not find command message for: " + input);
	}

	private ArrayList<Method> getCommandMessageMethod(Command command) {
		ArrayList<Method> methods = new ArrayList<>();

		for (Method m : this.getClass().getMethods()) {
			if (m.isAnnotationPresent(CommandMessage.class)) {
				String keyword = m.getAnnotation(CommandMessage.class).keyword();

				if (keyword.isEmpty()) {
					keyword = m.getName().toLowerCase();
				}

				if (command.supportsKeyword(keyword)) {
					methods.add(m);
				}
			}
		}

		return methods;
	}

	public String getName() {
		return this.getClass().getSimpleName();
	}

	public void onAnyMessage(Message message) {

	}

	public void onChannelMessage(Bot bot, String channel, String sender, String login, String hostname, String message) {
		this.onAnyMessage(new Message(bot, channel, sender, new Command(message)));
	}

	public void onPrivateMessage(Bot bot, String sender, String message) {
		this.onAnyMessage(new Message(bot, null, sender, new Command(message)));
	}

	public void onTimerTick(Bot bot, String channel) {}
}
