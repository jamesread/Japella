package japellaTest;

import japella.Main;
import japella.Server;
import japella.Server.NotFoundException;

import org.junit.Assert;
import org.junit.Test;

public class TestMain {
	public TestMain() throws NotFoundException {
		Server server = new Server("testingServer", "localhost", 9999);
		Main.instance = new Main();
		Main.instance.servers.add(server);

		Assert.assertEquals(server, Main.instance.lookupServer("testingServer"));
	}

	@Test
	public void testConfigDir() throws Exception {
		Assert.assertNotNull(Main.getConfigDir());
		Assert.assertTrue(Main.getConfigDir().exists());
	}

	@Test
	public void testConfigFile() throws Exception {
		Main main = new Main();

		Assert.assertNotNull(main.getConfigFile());
		Assert.assertTrue(main.getConfigFile().exists());
	}

	@Test(expected = Exception.class)
	public void testLookupNonexistantServer() throws Exception {
		Main.instance.lookupServer("I dont exist");
	}

}
