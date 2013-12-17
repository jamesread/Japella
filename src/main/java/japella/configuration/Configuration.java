package japella.configuration;

import japella.Bot;
import japella.Main;
import japella.MessagePlugin;
import japella.MessagePlugin.MessagePluginTimer;
import japella.Server;
import japella.Server.NotFoundException;

import java.io.File;
import java.io.IOException;
import java.util.Iterator;
import java.util.List;

import javax.xml.XMLConstants;
import javax.xml.transform.dom.DOMSource;
import javax.xml.transform.stream.StreamSource;
import javax.xml.validation.Schema;
import javax.xml.validation.SchemaFactory;
import javax.xml.validation.Validator;

import org.apache.commons.configuration.CompositeConfiguration;
import org.apache.commons.configuration.ConfigurationException;
import org.apache.commons.configuration.HierarchicalConfiguration;
import org.apache.commons.configuration.XMLConfiguration;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.xml.sax.SAXException;
import org.xml.sax.SAXParseException;

public class Configuration extends CompositeConfiguration {
	private static final transient Logger LOG = LoggerFactory.getLogger(Configuration.class);

	public String getVersion() {
		return "0.0.1";
	}

	private void parseBotChannelPlugins(Bot bot, String channelName, List<HierarchicalConfiguration> config) {
		for (HierarchicalConfiguration plugin : config) {
			MessagePlugin messagePlugin = bot.getMessagePlugin(plugin.getString("[@name]"));

			new MessagePluginTimer(bot, channelName, messagePlugin, plugin.getInt("[@period]"));

			bot.log("Bot " + bot.getName() + " will run " + messagePlugin.getClass().getSimpleName() + " in channel: " + channelName);
		}
	}

	private void parseBotChannels(Bot bot, List<HierarchicalConfiguration> list) {
		for (HierarchicalConfiguration channel : list) {
			String channelName = channel.getString("[@name]");

			bot.addChannel(channelName);

			this.parseBotChannelPlugins(bot, channelName, channel.configurationsAt("plugin"));
		}
	}

	private void parseBots(List<HierarchicalConfiguration> configurationsAt) {
		for (HierarchicalConfiguration bot : configurationsAt) {
			Configuration.LOG.info("Found bot: " + bot.getString("[@name]"));

			Server server = null;

			try {
				server = Main.instance.lookupServer(bot.getString("[@serverRef]"));
			} catch (NotFoundException e) {
				Configuration.LOG.warn("Server does not exist: " + bot.getString("[@serverRef]"));
				return;
			}

			Bot newBot = new Bot(bot.getString("[@name]"), server);
			newBot.setOwnerNickname(bot.getString("[@ownerNickname]"));
			newBot.setPassword(bot.getString("[@password]"));

			if (bot.containsKey("[@watchDirectory]")) {
				newBot.setWatchDirectory(new File(bot.getString("[@watchDirectory]")));
			}

			this.parseBotChannels(newBot, bot.configurationsAt("channel"));
			Main.instance.botList.add(newBot);

			newBot.start();
		}
	}

	public void parseConfigurationFile(File f) throws ConfigurationException {
		Configuration.LOG.info("Parsing configuration:" + f.getAbsolutePath() + ", exists: " + f.exists());

		try {
			XMLConfiguration botsFile = new NamespaceAwareXmlConfiguration(f);
			org.w3c.dom.Document d = botsFile.getDocument();
			SchemaFactory sf = SchemaFactory.newInstance(XMLConstants.W3C_XML_SCHEMA_NS_URI);

			StreamSource steamSource = new StreamSource(Configuration.class.getResourceAsStream("/configuration.xsd"));
			Schema s = sf.newSchema(steamSource);
			Validator val = s.newValidator();
			ConfigurationErrorHandler configErrorHandler = new ConfigurationErrorHandler(f);
			val.setErrorHandler(configErrorHandler);
			val.validate(new DOMSource(d));

			if (configErrorHandler.hasErrors()) {
				for (SAXParseException e : configErrorHandler.getParseErrors()) {
					Configuration.LOG.warn("Configuration parse error: " + e.getMessage());
				}

				throw new ConfigurationException(f.getName() + " has parse errors");
			}

			this.addConfiguration(botsFile);

			this.parseServers(botsFile.configurationsAt("server"));
			this.parseBots(botsFile.configurationsAt("bot"));
		} catch (SAXException e) {
			Configuration.LOG.error("Configuration exception:" + e.getMessage() + ". file: ", e);
		} catch (IOException e) {
			Configuration.LOG.error("IOException: " + e);
		}
	}

	private void parseServers(List<HierarchicalConfiguration> serverList) {
		for (HierarchicalConfiguration server : serverList) {
			Configuration.LOG.info("Found server!" + server.getString("[@name]"));

			Iterator<String> it = server.getKeys();

			while (it.hasNext()) {
				Configuration.LOG.info(it.next());
			}

			Server newServer = new Server(server.getString("[@name]"), server.getString("[@address]"), server.getInt("[@port]"));

			Main.instance.servers.add(newServer);
		}
	}
}
