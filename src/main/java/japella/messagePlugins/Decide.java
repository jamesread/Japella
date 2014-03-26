package japella.messagePlugins;

import japella.MessagePlugin;

import java.util.Random;

public class Decide extends MessagePlugin {

	@CommandMessage(keyword = "!decide")
	public void decide(Message message) {
		String thingToDo = message.parser.getBody();

		Random r = new Random();

		if (r.nextBoolean()) {
			message.reply("Yep, you should " + thingToDo);
		} else {
			message.reply("Nope, don't " + thingToDo);
		}
	}
}
