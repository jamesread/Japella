package japella.messagePlugins;

import japella.MessagePlugin;

import java.util.Random;

public class Decide extends MessagePlugin {

	@CommandMessage(keyword = "!decide")
	public void decide(Message msg) {
		String thingToDo = msg.originalMessage.replace("!decide", "").trim();

		Random r = new Random();

		if (r.nextBoolean()) {
			msg.reply("Yep, you should " + thingToDo);
		} else {
			msg.reply("Nope, don't " + thingToDo);
		}
	}
}
