package japella.messagePlugins;

import japella.MessagePlugin;

public class TestingPlugin extends MessagePlugin {

	@CommandMessage(keyword = "!goodbye")
	public void onGoodbye(Message message) {
		message.reply("aww, see you later :( ");
	}

	@CommandMessage(keyword = "!hello", target = MessageTarget.CHAT)
	public void onHello(Message message) {
		message.reply("Oh, hi there.");
	}
}
