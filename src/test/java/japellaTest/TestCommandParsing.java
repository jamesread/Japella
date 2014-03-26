package japellaTest;

import japella.MessageParser;

import org.junit.Assert;
import org.junit.Test;

public class TestCommandParsing {

	@Test
	public void testMessageParserTypes1() throws Exception {
		MessageParser parser = new MessageParser("!guess age 18");

		Assert.assertTrue(parser.matches(String.class, String.class, Integer.class));
		Assert.assertTrue(parser.hasParam(0));
		Assert.assertTrue(parser.hasParam(1));
		Assert.assertTrue(parser.hasParam(2));
		Assert.assertFalse(parser.hasParam(3));
		Assert.assertEquals("!guess", parser.getKeyword());
		Assert.assertEquals("!guess", parser.getString(0));
		Assert.assertEquals("age", parser.getString(1));
		Assert.assertEquals(18, parser.getInt(2));
	}

	@Test
	public void testParser1() {
		MessageParser parser = new MessageParser("!say this is a testing message");

		Assert.assertTrue(parser.hasKeyword("!say"));
		Assert.assertEquals("!say", parser.getKeyword());
		Assert.assertEquals("this is a testing message", parser.getBody());
	}

	@Test
	public void testParser2() {
		MessageParser parser = new MessageParser("!say one two three");

		Assert.assertTrue(parser.hasKeyword("!say"));
		Assert.assertEquals("!say", parser.getKeyword());
		Assert.assertEquals("one two three", parser.getBody());
		Assert.assertEquals("two three", parser.getBody(1));
		Assert.assertEquals("three", parser.getBody(2));
		Assert.assertEquals("", parser.getBody(9));
	}
}
