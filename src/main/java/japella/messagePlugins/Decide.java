package japella.messagePlugins;

import japella.MessagePlugin;

import java.util.Random;

public class Decide extends MessagePlugin {

	@CommandMessage(keyword = "!decide")
	public void decide(Message msg) {
		String message = msg.originalMessage;

		Random r = new Random();

		message = message.replace("!decide", "");
		message = message.trim();

		if (r.nextBoolean()) {
			msg.reply("Yep, you should " + message);
		} else {
			msg.reply("Nope, don't " + message);
		}
	}
}
