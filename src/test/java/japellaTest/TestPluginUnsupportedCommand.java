package japellaTest;

import japella.MessageParser;
import japella.MessagePlugin;
import japella.MessagePlugin.Message;

import org.junit.Assert;
import org.junit.Test;

public class TestPluginUnsupportedCommand {
	class BadPlugin extends MessagePlugin {
		@CommandMessage(keyword = "!test")
		public void methodWithBadSignature() {

		}

		@CommandMessage()
		public void methodWithBadSignatureNoKeyword() {

		}
	}

	@Test
	public void testBadMethod() {
		BadPlugin helpPlugin = new BadPlugin();
		boolean result = helpPlugin.callCommandMessages(new Message(null, "#test", "auser", new MessageParser("!test")));

		Assert.assertFalse(result);
	}

	@Test
	public void testUnsupportedCommand() {
		BadPlugin helpPlugin = new BadPlugin();
		boolean result = helpPlugin.callCommandMessages(new Message(null, "#test", "auser", new MessageParser("I have not commands for you")));

		Assert.assertFalse(result);
	}
}
