package japella.messagePlugins;

import japella.MessagePlugin;

import java.util.Random;

public class Decide extends MessagePlugin {
	public static Random randomGenerator = new Random();

	@CommandMessage(keyword = "!decide")
	public void decide(Message message) {
		String thingToDo = message.parser.getBody();

		if (Decide.randomGenerator.nextBoolean()) {
			message.reply("Yep, you should " + thingToDo);
		} else {
			message.reply("Nope, don't " + thingToDo);
		}
	}
}
