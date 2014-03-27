package japella;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.nio.file.FileSystems;
import java.nio.file.Path;
import java.nio.file.StandardWatchEventKinds;
import java.nio.file.WatchEvent;
import java.nio.file.WatchKey;
import java.nio.file.WatchService;
import java.util.Properties;
import java.util.Vector;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.locks.Condition;
import java.util.concurrent.locks.ReentrantLock;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class DirectoryMessageWatcher implements Runnable {
	private final AtomicBoolean run = new AtomicBoolean(true);

	private static final transient Logger LOG = LoggerFactory.getLogger(DirectoryMessageWatcher.class);

	private final static Vector<DirectoryMessageWatcher> registry = new Vector<DirectoryMessageWatcher>();

	public static void stopAll() {
		for (DirectoryMessageWatcher watcher : DirectoryMessageWatcher.registry) {
			watcher.stop();
		}
	}

	private WatchService watcher;

	private Bot bot;

	private final ReentrantLock lock = new ReentrantLock();

	private final Condition hasNewFile = this.lock.newCondition();

	public int numFiles = 0;

	public DirectoryMessageWatcher(Bot bot) throws IOException {
		Path directoryToWatch = bot.getWatchDirectory().toPath();

		if ((directoryToWatch == null) || !directoryToWatch.toFile().exists()) {
			return;
		}

		try {
			this.watcher = FileSystems.getDefault().newWatchService();
			this.bot = bot;

			directoryToWatch.register(this.watcher, StandardWatchEventKinds.ENTRY_CREATE);

			DirectoryMessageWatcher.registry.add(this);
			new Thread(this, "Directory watcher for " + bot.getNick()).start();
		} catch (Exception e) {

		}
	}

	private void parseFile(File f) throws IOException {
		if (!f.getName().endsWith(".msg")) {
			return;
		}

		FileInputStream fis = new FileInputStream(f);
		Properties properties = new Properties();
		properties.load(fis);
		fis.close();

		String channel = properties.getProperty("channel");
		String message = properties.getProperty("message");

		this.bot.debugMessage("Found new message: " + f.getName());
		this.bot.debugMessage("channel=" + channel);
		this.bot.debugMessage("message=" + message);

		if (!this.bot.isInChannel(channel)) {
			this.bot.debugMessage("Cannot send message to a channel that I'm not in: " + channel);
		}

		this.bot.sendMessageResponsibly(channel.trim(), message.trim());

		f.delete();

		this.numFiles++;
	}

	@Override
	public void run() {

		try {
			while (this.run.get()) {
				WatchKey key = this.watcher.take();

				DirectoryMessageWatcher.LOG.debug("watching");

				for (WatchEvent<?> rawEvt : key.pollEvents()) {
					this.lock.lock();

					DirectoryMessageWatcher.LOG.debug("handling event:" + rawEvt.toString());

					WatchEvent<Path> evt = (WatchEvent<Path>) rawEvt;
					Path filename = evt.context();

					DirectoryMessageWatcher.LOG.debug("event context: " + filename);

					DirectoryMessageWatcher.LOG.info(filename.toFile().getAbsolutePath());
					File newMessage = new File(this.bot.getWatchDirectory(), filename.toFile().getName());

					DirectoryMessageWatcher.LOG.info(newMessage.getAbsolutePath());

					if (newMessage.exists()) {
						this.parseFile(newMessage);

						this.hasNewFile.signalAll();
					}
					this.lock.unlock();
				}
			}
		} catch (Exception e) {
			DirectoryMessageWatcher.LOG.error("error watching directory", e);
		} finally {
			this.lock.unlock();
		}
	}

	public void stop() {
		this.run.set(false);
	}

	public void waitForNewFile() throws InterruptedException {
		this.lock.lock();

		this.hasNewFile.await(3, TimeUnit.SECONDS);

		this.lock.unlock();
	}
}
