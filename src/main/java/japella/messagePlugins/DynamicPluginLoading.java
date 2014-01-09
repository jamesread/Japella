package japella.messagePlugins;

import japella.MessagePlugin;

public class DynamicPluginLoading extends MessagePlugin {
	@CommandMessage(keyword = "!pluginToggle")
	public void pluginToggle(Message message) {
		if (message.fromAdmin()) {
			message.reply("Cannot toggle plugins yet, feature not implemented.");
		} else {
			message.reply("You cannot toggle plugins, you are not an admin.");
		}
	}
}
