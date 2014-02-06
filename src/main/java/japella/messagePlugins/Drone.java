package japella.messagePlugins;

import japella.MessagePlugin;

public class Drone extends MessagePlugin {

	@CommandMessage(keyword = "!say", target = MessageTarget.PM)
	public void onSay(Message message) {
		String channel = message.command.getString(1);
		String thingToSay = message.command.getString(2);

		if (message.bot.isInChannel(channel)) {
			message.bot.sendMessageResponsibly(channel, thingToSay);
			message.reply("Sent to: " + channel);
		} else {
			message.reply("I'm not in the channel: " + channel);
		}
	}
}
