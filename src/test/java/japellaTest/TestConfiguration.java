package japellaTest;

import japella.Main;
import japella.configuration.Configuration;

import java.io.File;

import org.apache.commons.configuration.ConfigurationException;
import org.junit.Assert;
import org.junit.Test;

public class TestConfiguration {
	@Test
	public void testConfigurationVersion() {
		Configuration configuration = new Configuration();

		Assert.assertNotNull(Configuration.getVersion());
	}

	@Test
	public void testParseTestConfiguration() throws ConfigurationException {
		Main.instance = new Main();
		Configuration configuration = new Configuration();

		configuration.parseConfigurationFile(new File("src/test/resources/testingConfiguration.xml"));
	}
}
