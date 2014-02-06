package japella.configuration;

import japella.Main;
import japella.MessagePlugin;

import java.io.File;
import java.util.HashMap;
import java.util.Properties;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;


import org.apache.commons.configuration.PropertiesConfiguration;

public class PropertiesFileCollection extends Properties {
	private static final transient Logger LOG = LoggerFactory.getLogger(Main.class);
	private static final HashMap<String, PropertiesConfiguration> db = new HashMap<>();

	public static PropertiesConfiguration get(MessagePlugin p) throws Exception {
		return PropertiesFileCollection.get(p.getName() + ".properties");
	}

	public static PropertiesConfiguration get(String fileName) throws Exception {
		PropertiesFileCollection.load(fileName);

		return PropertiesFileCollection.db.get(fileName);
	}

	public static void load(String fileName) throws Exception {
		LOG.info("Trying to load properties file: " + fileName + " from " + (new File(Main.getConfigDir(), fileName)).getAbsolutePath());

		if (!PropertiesFileCollection.db.containsKey(fileName)) {
			LOG.info("File hasn't been cached, first load of " + fileName);

			File file = new File(Main.getConfigDir(), fileName);

			if (!file.exists()) {
				file.createNewFile();
			}

			PropertiesConfiguration cfg = new PropertiesConfiguration(file);
			PropertiesFileCollection.db.put(fileName, cfg);
		}
	}
}
