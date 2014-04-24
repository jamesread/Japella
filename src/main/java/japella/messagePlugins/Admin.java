package japella.messagePlugins;

import japella.Main;
import japella.MessagePlugin;

public class Admin extends MessagePlugin {
	@CommandMessage(keyword = "!join")
	public void join(Message message) {
		String channel = message.parser.getStringFirstArgument();

		if (message.fromAdmin()) {
			message.bot.joinChannel(channel);
		} else {
			message.reply("You cannot tell me to join a channel (" + channel + "), because you are not an admin");
		}

	}

	@CommandMessage(keyword = "!part")
	public void part(Message message) {
		String channel = message.parser.getStringFirstArgument();

		if (message.fromAdmin()) {
			message.reply("Goodbye, I'm quitting!");

			message.bot.partChannel(channel);
		} else {
			message.reply("You cannot tell me to leave a channel (" + channel + "), because you are not an admin");

		}
	}

	@CommandMessage(keyword = "!password")
	public void password(Message message) {
		final String password = message.parser.getStringFirstArgument();
		final String whoisLine = message.bot.getWhois(message.sender);

		if (whoisLine == null) {
			message.reply("Cannot get your WHOIS details. Did you do a !whoami ?");

			return;
		}

		if (message.bot.isAdminPassword(password)) {
			message.reply("Password accepted.");
			message.bot.addAdmin(message.sender);
		} else {
			message.reply("Password rejected");
		}
	}

	@CommandMessage(keyword = "!quit")
	public void quit(Message message) {
		if (message.fromAdmin()) {
			Main.instance.shutdown();
		} else {
			message.reply("You are not an admin, so I won't quit.");
		}
	}

}
