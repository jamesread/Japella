package japella.messagePlugins;

import japella.MessagePlugin;

public class GaggingPlugin extends MessagePlugin {
	@CommandMessage(keyword = "!sleep")
	public void quiet(Message message) {
		message.bot.addChannelGag(message.channel);
	}

	@CommandMessage(keyword = "!wakeup")
	public void wakeup(Message message) {
		message.bot.removeChannelGag(message.channel);
	}
}
