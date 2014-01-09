package japella.messagePlugins;

import japella.Bot;
import japella.MessagePlugin;

import java.util.Vector;

public class GaggingPlugin extends MessagePlugin {
	private static Vector<String> quietCommands = new Vector<String>();

	static {
		GaggingPlugin.quietCommands.add("quiet");
		GaggingPlugin.quietCommands.add("shutup");
		GaggingPlugin.quietCommands.add("silence");
		GaggingPlugin.quietCommands.add("pipe down");
	}

	@Override
	public String getName() {
		// TODO Auto-generated method stub
		return null;
	}

	private boolean isGagCommand(String message) {
		for (String gagCommand : GaggingPlugin.quietCommands) {
			if (message.contains(gagCommand)) {
				return true;
			}
		}

		return false;
	}

	@Override
	public void onChannelMessage(Bot bot, String channel, String sender, String login, String hostname, String message) {
		if (sender.equals(bot.getName())) {
			return;
		}

		if (!message.contains(bot.getName())) {
			return;
		}

		if (this.isGagCommand(message)) {
			bot.toggleChannelGag(channel);
		}
	}

	@Override
	public void onPrivateMessage(Bot bot, String sender, String message) {
		// TODO Auto-generated method stub

	}

}
