package japella.messagePlugins;

import japella.MessagePlugin;

public class Help extends MessagePlugin {
	@CommandMessage(keyword = "!admins")
	public void onAdmins(Message message) {
		message.reply(message.bot.getAdmins().toString());
	}

	@CommandMessage(keyword = "!help")
	public void onHelp(Message message) {
		String ret = "Hi. I support these commands: ";

		for (MessagePlugin plugin : message.bot.getMessagePlugins()) {
			for (CommandMessage command : plugin.getCommandMessages().keySet()) {
				ret += command.keyword() + " ";
			}
		}

		message.reply(ret);
	}

	@CommandMessage(keyword = "!plugins")
	public void onPlugins(Message message) {
		StringBuilder buf = new StringBuilder("Plugins: ");

		for (MessagePlugin mp : message.bot.getMessagePlugins()) {
			buf.append(mp.getClass().getSimpleName() + ". ");
		}

		message.reply(buf.toString());
	}

	@CommandMessage(keyword = "!version")
	public void onVersion(Message message) {
		message.reply("?.?.?");
	}
}