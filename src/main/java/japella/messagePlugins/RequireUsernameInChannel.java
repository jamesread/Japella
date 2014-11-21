package japella.messagePlugins;

import japella.Bot;
import japella.MessagePlugin;
import japella.configuration.PropertiesFileCollection;

import org.apache.commons.configuration.PropertiesConfiguration;
import org.jibble.pircbot.User;

public class RequireUsernameInChannel extends MessagePlugin {
	private PropertiesConfiguration config;

	public RequireUsernameInChannel() {
		try {
			this.config = PropertiesFileCollection.get(this);
		} catch (Exception e) {
			e.printStackTrace();
		}
	}

	@Override
	public void onTimerTick(Bot bot, String channel) {
		User[] userList = bot.getUsers(channel);

		String requiredUsernameSuffix = this.config.getString("requiredUsernameSuffix").toLowerCase();

		for (User user : userList) {
			if (user.getNick().toLowerCase().endsWith(requiredUsernameSuffix)) {
				return;
			}
		}

		bot.sendMessageResponsibly(channel, "There is no active gatekeeper on IRC! (The gatekeeper username needs to end with '" + requiredUsernameSuffix + "'");
	}
}
