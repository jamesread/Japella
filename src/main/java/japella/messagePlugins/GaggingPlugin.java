package japella.messagePlugins;

import japella.MessagePlugin;

import java.util.Map.Entry;

import org.joda.time.Duration;
import org.joda.time.Instant;
import org.joda.time.Interval;

public class GaggingPlugin extends MessagePlugin {
	@CommandMessage(keyword = "!gags")
	public void gags(Message message) {
		if (message.bot.hasGags()) {
			for (Entry<String, Instant> gag : message.bot.getGags()) {
				Instant timeout = gag.getValue();
				String stimeout;

				if (timeout == null) {
					stimeout = " for forever.";
				} else {
					stimeout = " for " + (new Interval(timeout, Instant.now()).toDuration().toPeriod()) + ", until " + timeout + " now = " + Instant.now();
				}

				message.bot.sendMessage(message.channel, "Gagged in " + gag.getKey() + stimeout + "\n");
			}
		} else {
			message.reply("I'm not gagged in any channel, I'm as free as a bird!");
		}
	}

	@CommandMessage(keyword = "!hibernate")
	public void hibernate(Message message) {
		message.bot.addChannelGag(message.channel, null);
	}

	@CommandMessage(keyword = "!nap")
	public void nap(Message message) {
		message.bot.addChannelGag(message.channel, Duration.standardMinutes(40));
	}

	@CommandMessage(keyword = "!sleep")
	public void sleep(Message message) {
		message.bot.addChannelGag(message.channel, Duration.standardHours(2));
	}

	@CommandMessage(keyword = "!wakeup")
	public void wakeup(Message message) {
		message.bot.removeChannelGag(message.channel);
	}
}
