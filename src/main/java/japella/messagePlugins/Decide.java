package japella.messagePlugins;

import japella.Bot;
import japella.MessagePlugin;

import java.util.Random;

public class Decide extends MessagePlugin {
	@Override
	public void addMessage(final String m) {

	}

	@CommandMessage
	public void decide() {

	}

	@Override
	public String getName() {
		return this.getClass().getSimpleName();
	}

	@Override
	public void onMessage(Bot bot, String channel, String sender, String login, String hostname, String message) {
		Random r = new Random();

		if (message.contains("!decide")) {
			message = message.replace("!decide", "");
			message = message.trim();

			if (r.nextBoolean()) {
				bot.sendMessageResponsibly(channel, "Yep, you should " + message);
			} else {
				bot.sendMessageResponsibly(channel, "Nope, don't " + message);
			}
		}
	}

	@Override
	public void onPrivateMessage(Bot bot, String sender, String message) {
		this.onMessage(bot, sender, sender, null, null, message);
	}

	@Override
	public void onTimerTick(Bot bot, String channel) {}
}
