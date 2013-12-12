package japella.messagePlugins;

import japella.Bot;
import japella.MessagePlugin;

public class DynamicPluginLoading extends MessagePlugin {

	@Override
	public void addMessage(String m) {
		// TODO Auto-generated method stub
	}

	@Override
	public void onMessage(Bot bot, String channel, String sender, String login, String hostname, String message) {
		if (message.contains("!pluginToggle")) {
			if (bot.hasAdmin(sender)) { 
				bot.sendMessageResponsiblyUser(channel, sender, "Cannot toggle plugins yet, feature not implemented.");
			} else {
				bot.sendMessageResponsiblyUser(channel, sender, "You cannot toggle plugins, you are not an admin.");
			}
		}
	}

	@Override
	public void onTimerTick(Bot bot, String channel) {
		// TODO Auto-generated method stub
	}

	@Override
	public void onPrivateMessage(Bot bot, String sender, String message) {
		// TODO Auto-generated method stub
	}

}
