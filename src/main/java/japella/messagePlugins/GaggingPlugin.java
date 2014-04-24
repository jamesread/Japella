package japella.messagePlugins;

import japella.MessagePlugin;

import org.joda.time.Duration;

public class GaggingPlugin extends MessagePlugin {
	@CommandMessage(keyword = "!quiet")
	public void quiet(Message message) {
		message.bot.addChannelGag(message.channel, null);
	}

	@CommandMessage(keyword = "!sleep")
	public void sleep(Message message) {
		message.bot.addChannelGag(message.channel, Duration.standardHours(2));
	}

	@CommandMessage(keyword = "!wakeup")
	public void wakeup(Message message) {
		message.bot.removeChannelGag(message.channel);
	}
}
