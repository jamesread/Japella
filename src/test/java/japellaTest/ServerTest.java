package japellaTest;

import japella.Server;

import org.junit.Assert;
import org.junit.Test;

public class ServerTest {
	@Test
	public void testServer() {
		Server server = new Server("test.com");

		Assert.assertEquals("test.com", server.getAddress());
		Assert.assertEquals(6667, server.getPort());
		Assert.assertEquals("???", server.getServerName());
	}

	@Test
	public void testServer2() {
		Server server = new Server("testName", "test.com", 999);

		Assert.assertEquals("test.com", server.getAddress());
		Assert.assertEquals(999, server.getPort());
		Assert.assertEquals("testName", server.getServerName());
	}
}
