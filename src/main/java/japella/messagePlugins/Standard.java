package japella.messagePlugins;

import japella.Main;
import japella.MessagePlugin;

import org.joda.time.Instant;
import org.joda.time.Interval;

public class Standard extends MessagePlugin {
	@CommandMessage(keyword = "!channels")
	public void foo(Message message) {
		String reply = "";

		for (String channel : message.bot.getChannels()) {
			reply += channel + " ";
		}

		message.reply(reply);
	}

	@CommandMessage(keyword = "!uptime")
	public void status(Message message) {
		message.reply("Uptime: " + new Interval(Main.instance.startTime, Instant.now()).toDuration().toPeriod().toString());
	}
}
