package japellaTest;

import japella.Bot;
import japella.DirectoryMessageWatcher;

import java.io.File;
import java.io.FileWriter;
import java.io.IOException;
import java.nio.file.Files;

import org.junit.Assert;
import org.junit.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class TestDirectoryWatcher {
	private final Logger LOG = LoggerFactory.getLogger(TestDirectoryWatcher.class);

	@Test
	public void test() throws IOException, InterruptedException {
		File watchDirectory = Files.createTempDirectory(null).toFile();
		watchDirectory.deleteOnExit();

		Assert.assertTrue(watchDirectory.exists());

		Bot bot = new Bot("watcher", null);
		bot.joinChannel("#test");
		DirectoryMessageWatcher watcher = bot.setWatchDirectory(watchDirectory);
		Assert.assertEquals(0, watcher.numFiles);

		File f = File.createTempFile("testWatchDirectory", ".msg", watchDirectory);
		FileWriter writer = new FileWriter(f);
		writer.write("channel=#test\n");
		writer.write("message=This is a test\n");
		writer.close();

		Assert.assertTrue(f.exists());

		watcher.waitForNewFile();

		Assert.assertFalse(f.exists());

		Assert.assertEquals(1, watcher.numFiles);
	}
}
