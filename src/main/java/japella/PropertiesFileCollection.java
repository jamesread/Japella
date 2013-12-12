package japella;

import java.io.File;
import java.util.HashMap;
import java.util.Properties;

import org.apache.commons.configuration.PropertiesConfiguration;

public class PropertiesFileCollection extends Properties {
	private static final HashMap<String, PropertiesConfiguration> db = new HashMap<>();

	public static PropertiesConfiguration get(MessagePlugin p) throws Exception {
		return PropertiesFileCollection.get(p.getName());
	}

	public static PropertiesConfiguration get(String fileName) throws Exception {
		PropertiesFileCollection.load(fileName);

		return PropertiesFileCollection.db.get(fileName);
	}

	public static void load(String fileName) throws Exception {
		if (!PropertiesFileCollection.db.containsKey(fileName)) {
			File file = new File(Main.getConfigDir(), fileName);

			if (!file.exists()) {
				file.createNewFile();
			}

			PropertiesConfiguration cfg = new PropertiesConfiguration(file);
			PropertiesFileCollection.db.put(fileName, cfg);
		}
	}
}
