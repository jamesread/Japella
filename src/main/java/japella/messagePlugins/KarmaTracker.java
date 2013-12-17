package japella.messagePlugins;

import japella.Bot;
import japella.MessagePlugin;
import japella.configuration.PropertiesFileCollection;

import java.util.Collections;
import java.util.Comparator;
import java.util.HashMap;
import java.util.Iterator;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;

import org.apache.commons.configuration.PropertiesConfiguration;
import org.joda.time.Duration;
import org.joda.time.Instant;
import org.joda.time.Interval;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class KarmaTracker extends MessagePlugin {
	public HashMap<String, Integer> karma = new HashMap<String, Integer>();
	private final HashMap<String, HashMap<String, Instant>> karmaLog = new HashMap<>();

	private final Duration karmaOverflowDelay = Duration.standardMinutes(20);

	private static final transient Logger LOG = LoggerFactory.getLogger(KarmaTracker.class);

	public KarmaTracker() {
		this.loadConfig();
	}

	@Override
	public void addMessage(String m) {}

	@Override
	public String getName() {
		return this.getClass().getSimpleName();
	}

	private void loadConfig() {
		try {
			PropertiesConfiguration properties = PropertiesFileCollection.get(this);
			properties.clear();
			properties.load();
			Iterator<String> keys = properties.getKeys();

			while (keys.hasNext()) {
				String k = keys.next();
				this.karma.put(k, properties.getInt(k));
			}
		} catch (Exception e) {
			KarmaTracker.LOG.error("Error during load", e);
		}
	}

	@Override
	public void onMessage(Bot bot, String channel, String sender, String login, String hostname, String message) {
		int maxToDisplay = Math.min(5, this.karma.size());
		if (message.equals("!rank")) {
			bot.sendMessageResponsibly(channel, "Karma ranks, top " + maxToDisplay + " (" + this.karma.size() + " in total):\n ");

			List<Map.Entry<String, Integer>> list = new LinkedList<Map.Entry<String, Integer>>(this.karma.entrySet());
			Collections.sort(list, new Comparator<Map.Entry<String, Integer>>() {
				@Override
				public int compare(Map.Entry<String, Integer> one, Map.Entry<String, Integer> two) {
					return (two.getValue()).compareTo(one.getValue());
				}
			});

			int count = 0;
			for (Map.Entry<String, Integer> row : list) {
				if (count > maxToDisplay) {
					break;
				} else {
					count++;
				}

				bot.sendMessageResponsibly(channel, row.getKey() + " (" + row.getValue() + " points)");
			}
		} else if (message.equals("!karmasave")) {
			bot.sendMessageResponsibly(channel, "Karma saved to file.");
			this.saveConfig();
		} else if (message.equals("!karmaload")) {
			bot.sendMessageResponsibly(channel, "Karma loaded from file.");
			this.loadConfig();
		}

		this.tryGiveKarma(bot, message, channel, sender);
	}

	@Override
	public void onPrivateMessage(Bot bot, String sender, String message) {}

	@Override
	public void onTimerTick(Bot bot, String channel) {
		KarmaTracker.LOG.debug("Karma saved on timer tick.");
		this.saveConfig();
	}

	public void saveConfig() {
		try {
			PropertiesConfiguration properties = PropertiesFileCollection.get(this);

			for (Entry<String, Integer> karma : this.karma.entrySet()) {
				properties.setProperty(karma.getKey(), karma.getValue());
			}

			properties.save();
		} catch (Exception e) {
		}
	}

	private void tryGiveKarma(Bot bot, String message, String channel, String sender) {
		int karmaDelta = 0;

		if (message.contains("++")) {
			karmaDelta++;
		} else if (message.contains("--")) {
			karmaDelta--;
		} else {
			return;
		}

		String thingToKarma = message.replace("++", "").replace("--", "").trim();

		if ((thingToKarma == null) || thingToKarma.isEmpty() || !Character.isAlphabetic(thingToKarma.charAt(0))) {
			return;
		}

		if (thingToKarma.equalsIgnoreCase(sender)) {
			bot.sendMessageResponsiblyUser(channel, sender, "You cannot change karma on yourself.");
			return;
		}

		if (this.karmaLog.get(sender) == null) {
			this.karmaLog.put(sender, new HashMap<String, Instant>());
		}

		HashMap<String, Instant> usersKarmaLog = this.karmaLog.get(sender);

		if ((usersKarmaLog != null) && usersKarmaLog.containsKey(thingToKarma)) {
			Instant lastKarmared = usersKarmaLog.get(thingToKarma);

			if (lastKarmared.plus(this.karmaOverflowDelay).isAfter(Instant.now())) {
				bot.sendMessageResponsiblyUser(channel, sender, "You must wait until " + new Interval(lastKarmared.plus(this.karmaOverflowDelay), Instant.now()).toDuration().getStandardMinutes() + " minute(s) until you can set karma on that again. ");
				return;
			}
		}

		if (this.karma.containsKey(thingToKarma)) {
			karmaDelta = this.karma.get(thingToKarma) + karmaDelta;
		}

		if (karmaDelta == 0) {
			this.karma.remove(thingToKarma);

			bot.sendMessageResponsibly(channel, thingToKarma + " now has 0 points and has been garbage collected.");
		} else {
			this.karma.put(thingToKarma, karmaDelta);
			bot.sendMessageResponsibly(channel, thingToKarma + " now has " + karmaDelta + " karma");
		}

		usersKarmaLog.put(thingToKarma, Instant.now());
	}
}
