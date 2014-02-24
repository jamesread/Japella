package japella.messagePlugins;

import japella.MessagePlugin;

public class Help extends MessagePlugin {
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
}
