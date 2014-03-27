package japella;

import java.io.File;
import java.io.FileInputStream;
import java.util.Properties;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class DirectoryMessageWatcher implements Runnable {
	private boolean run = true;
	private static final transient Logger LOG = LoggerFactory.getLogger(DirectoryMessageWatcher.class);

	@Override
	public void run() {
		while (this.run) {
			try {
				Thread.sleep(10000);

				for (Bot bot : Main.instance.botList) {
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

							if (!bot.isInChannel(channel)) {
								bot.debugMessage("Cannot send message to a channel that I'm not in: " + channel);
							}

							bot.sendMessageResponsibly(channel.trim(), message.trim());

							f.delete();
						}
					}
				}
			} catch (Exception e) {
				DirectoryMessageWatcher.LOG.error("Exception in DirectoryMessageWatcher: " + e.toString(), e);
			}
		}
	}

	public void start() {
		new Thread(this).start();
	}

	public void stop() {
		this.run = false;
	}
}
