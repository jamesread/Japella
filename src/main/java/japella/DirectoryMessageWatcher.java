package japella;

import java.io.File;
import java.io.FileInputStream;
import java.util.Properties;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class DirectoryMessageWatcher implements Runnable {
	private boolean run = true;
	private Main main;
	private static final transient Logger LOG = LoggerFactory.getLogger(DirectoryMessageWatcher.class);

	private boolean isInChannel(Bot bot, String channel) {
		for (String joinedChannel : bot.getChannels()) {
			if (joinedChannel.equals(channel)) {
				return true;
			}
		}

		return false;
	}

	@Override
	public void run() {
		while (this.run) {
			try {
				Thread.sleep(10000);

				for (Bot bot : this.main.botList) {
					File directoryToWatch = bot.getWatchDirectory();

					if ((directoryToWatch == null) || !directoryToWatch.exists()) {
						continue;
					}

					File[] messages = directoryToWatch.listFiles();

					for (File f : messages) {
						if (f.getName().endsWith(".msg")) {
							FileInputStream fis = new FileInputStream(f);
							Properties properties = new Properties();
							properties.load(fis);
							fis.close();

							String channel = properties.getProperty("channel");
							String message = properties.getProperty("message");

							bot.debugMessage("Found new message: " + f.getName());
							bot.debugMessage("channel=" + channel);
							bot.debugMessage("cl:" + channel.length());
							bot.debugMessage("message=" + message);

							if (!this.isInChannel(bot, channel)) {
								bot.debugMessage("Cannot send message to a channel that I'm not in: " + channel);
							}

							bot.sendMessageResponsibly(channel.trim(), message.trim());

							f.delete();
						}
					}
				}
			} catch (Exception e) {
				LOG.error("Exception in DirectoryMessageWatcher: " + e.toString(), e);
			}
		}
	}

	public void setMain(Main main) {
		this.main = main;
	}

	public void start() {
		new Thread(this).start();
	}

	public void stop() {
		this.run = false;
	}
}
