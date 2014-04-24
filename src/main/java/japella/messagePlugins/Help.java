package japella.messagePlugins;

import japella.MessagePlugin;

import java.util.Collections;
import java.util.Vector;

public class Help extends MessagePlugin {
	@CommandMessage(keyword = "!admins")
	public void onAdmins(Message message) {
		message.reply(message.bot.getAdmins().toString());
	}

	@CommandMessage(keyword = "!help")
	public void onHelp(Message message) {
		String ret = "Hi. I support these commands: ";
		Vector<String> supportedCommands = new Vector<String>();

		for (MessagePlugin plugin : message.bot.getMessagePlugins()) {
			for (CommandMessage command : plugin.getCommandMessages().keySet()) {
				supportedCommands.add(command.keyword());
			}
		}

		Collections.sort(supportedCommands);

		ret += supportedCommands.toString();

		message.reply(ret.trim());
	}

	@CommandMessage(keyword = "!plugins")
	public void onPlugins(Message message) {
		StringBuilder buf = new StringBuilder("Plugins: ");

		Vector<String> pluginNames = new Vector<String>();

		for (MessagePlugin mp : message.bot.getMessagePlugins()) {
			pluginNames.add(mp.getClass().getSimpleName());
		}

		Collections.sort(pluginNames);

		message.reply("Plugins: " + pluginNames.toString());
	}

	@CommandMessage(keyword = "!version")
	public void onVersion(Message message) {
		message.reply("?.?.?");
	}
}