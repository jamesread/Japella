package japella;

import japella.MessagePlugin.Message;
import japella.messagePlugins.Decide;
import japella.messagePlugins.Drone;
import japella.messagePlugins.GaggingPlugin;
import japella.messagePlugins.HelloWorld;
import japella.messagePlugins.Help;
import japella.messagePlugins.KarmaTracker;
import japella.messagePlugins.QuizPlugin;
import japella.messagePlugins.TicketLookup;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileReader;
import java.util.ArrayList;
import java.util.Hashtable;
import java.util.Vector;

import org.jibble.pircbot.NickAlreadyInUseException;
import org.jibble.pircbot.PircBot;
import org.jibble.pircbot.ReplyConstants;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class Bot extends PircBot implements Runnable {
	private String nick = null;
	private Server server = null;
	private boolean operator = false;
	private final Vector<String> channels = new Vector<String>();
	private String password = "supersecret";
	private final Vector<String> admins = new Vector<String>();

	private String ownerNickname = "unknownOwner";
	private final ArrayList<MessagePlugin> messagePlugins = new ArrayList<MessagePlugin>();

	private final Thread runner;

	private String lastWhois = "";

	private static final transient Logger LOG = LoggerFactory.getLogger(Bot.class);

	private final Hashtable<String, String> cachedWhoisQueries = new Hashtable<String, String>();

	private File watchDirectory;

	private final Vector<String> channelGags = new Vector<String>();

	public Bot(final String nick, final Server server) {
		this.nick = nick;
		this.setVerbose(true);

		this.debugMessage("Constructing bot: " + this.nick + " with password: " + this.password);

		this.setName(this.nick);
		this.setVersion("Japella");
		this.setFinger("Get your fingers off me!");
		this.setLogin(this.nick);

		this.server = server;

		this.runner = new Thread(this, "Bot: " + this.nick);
	}

	public void addAdmin(String admin) {
		this.admins.add(admin);
	}

	public void addChannel(String channel) {
		this.channels.add(channel);
	}

	public void addChannelGag(String channel) {
		this.channelGags.add(channel);
		this.log("Channel gag added on channel: " + channel);
		this.sendMessage(channel, "I will no longer send messages to this channel.");
	}

	private void connect() {
		try {
			this.debugMessage("Attempting to connect to server \"" + this.server.getServerName() + "\".\n");
			this.connect(this.server.getAddress(), this.server.getPort());
			this.debugMessage("Connected to \"" + this.server.getServerName() + "\"");

			this.joinAllChannels();
		} catch (final NickAlreadyInUseException e) {
			this.debugMessage("Nickname already in use on \"" + this.server.getServerName() + "\".\n");

			this.disconnect();
		} catch (final Exception e) {
			this.debugMessage("Cannot connect: " + e.toString() + "\n");
		}
	}

	public void debugMessage(final String message) {
		Bot.LOG.debug(this.nick + ": " + message.trim());
	}

	public Vector<String> getAdmins() {
		return this.admins;
	}

	public MessagePlugin getMessagePlugin(String pluginName) {
		for (MessagePlugin mp : this.messagePlugins) {
			if (mp.getClass().getSimpleName().equals(pluginName)) {
				return mp;
			}
		}

		return null;
	}

	public ArrayList<MessagePlugin> getMessagePlugins() {
		return this.messagePlugins;
	}

	public File getWatchDirectory() {
		return this.watchDirectory;
	}

	public String getWhois(final String username) {
		if (this.cachedWhoisQueries.containsKey(username)) {
			return this.cachedWhoisQueries.get(username);
		}

		this.debugMessage("Got reply: " + this.lastWhois);
		this.cachedWhoisQueries.put(username, this.lastWhois);

		return this.lastWhois;
	}

	public boolean hasAdmin(String sender) {
		return this.admins.contains(sender);
	}

	public boolean isGagged(String channel) {
		return this.channelGags.contains(channel);
	}

	public boolean isInChannel(String channel) {
		return this.channels.contains(channel);
	}

	private void joinAllChannels() {
		this.debugMessage("Going to join " + this.channels.size() + " channels.");

		String currentChannel = "";

		for (int i = 0; i < this.channels.size(); i++) {
			currentChannel = this.channels.get(i);

			this.joinChannel(currentChannel);
		}

	}

	public void loadDefaultMessagePlugins() {
		this.messagePlugins.add(new KarmaTracker());
		this.messagePlugins.add(new Decide());
		this.messagePlugins.add(new TicketLookup());
		this.messagePlugins.add(new GaggingPlugin());
		this.messagePlugins.add(new QuizPlugin());
		this.messagePlugins.add(new HelloWorld());
		this.messagePlugins.add(new Help());
		this.messagePlugins.add(new Drone());
	}

	public void loadMessagePlugin(MessagePlugin plugin) {
		this.messagePlugins.add(plugin);
	}

	private final Message onAnyMessage(Message message) {
		for (MessagePlugin mp : this.messagePlugins) {
			if (message.originalMessage.startsWith("!")) {
				mp.callCommandMessages(message);
			}
		}

		return message;
	}

	@Override
	public void onDeop(final String channel, final String sourceNick, final String sourceLogin, final String sourceHostname, final String recipient) {
		if (recipient.matches(this.nick)) {
			this.debugMessage("De-opped in \"" + channel + "\" by \"" + sourceNick + "\".\n");

			this.operator = false;
		}
	}

	@Override
	public void onJoin(final String channel, final String sender, final String login, final String hostname) {
		if (this.nick.equals(sender)) {
			this.debugMessage("I have joined \"" + channel + "\".");
		} else {
			this.debugMessage("\"" + sender + "\" joined \"" + channel + "\".");
		}

		this.setUserModes(channel, sender, hostname);
	}

	@Override
	public void onMessage(final String channel, final String sender, final String login, final String hostname, String smessage) {
		if (smessage.startsWith(this.getName())) {
			smessage = smessage.replaceFirst(this.getName() + ": ", "");
		}

		Message message = this.onAnyMessage(new Message(this, channel, sender, new MessageParser(smessage)));

		for (MessagePlugin mp : this.messagePlugins) {
			mp.onChannelMessage(message);
		}
	}

	public Message onMockMessage(Message message) {
		return this.onAnyMessage(message);
	}

	@Override
	public void onOp(final String channel, final String sourceNick, final String sourceLogin, final String sourceHostname, final String recipient) {
		if (recipient.matches(this.nick)) {
			this.debugMessage("Op'ed in \"" + channel + "\" by \"" + sourceNick + "\".");
			this.operator = true;
		}
	}

	/**
	 * Called by the superclass when the bot recieves a private message.
	 */
	@Override
	public void onPrivateMessage(final String sender, final String login, final String hostname, final String message) {
		Bot.LOG.debug("recv PM: " + sender + ": " + message);

		this.onAnyMessage(new Message(this, null, sender, new MessageParser(message)));

		if (message.contains("!join")) {
			String channel = message.replace("!join", "").trim();

			if (this.admins.contains(sender)) {
				this.joinChannel(channel);
			} else {
				String reply = "You cannot tell me to join a channel (" + channel + "), because you are not an admin";

				this.sendMessageResponsibly(sender, reply);
				this.log(reply);
			}
		} else if (message.startsWith("!channels")) {
			String reply = "";

			for (String channel : this.getChannels()) {
				reply += channel + " ";
			}

			this.sendMessageResponsibly(sender, reply);
			this.log(reply);
		} else if (message.contains("!part")) {
			String channel = message.replace("!part", "").trim();

			if (this.admins.contains(sender)) {
				this.partChannel(channel);
			} else {
				String reply = "You cannot tell me to leave a channel (" + channel + "), because you are not an admin";

				this.sendMessageResponsibly(sender, reply);
				this.log(reply);
			}
		} else if (message.startsWith("!quit")) {
			if (this.admins.contains(sender)) {

				Main.instance.shutdown();
			} else {
				this.sendMessageResponsibly(sender, "You are not an admin. Won't quit.");
			}
		} else if (message.contains("!password")) {
			final String password = message.replace("!password ", "");

			final String whoisLine = this.getWhois(sender);

			if (whoisLine == null) {
				this.sendMessageResponsibly(sender, "Cannot get your WHOIS details. Did you do a !whoami ?");
				this.debugMessage("Cannot get WHOIS details for " + sender);

				return;
			}

			if (password.trim().equals(this.password)) {
				this.sendMessageResponsibly(sender, "Password accepted.");
				this.debugMessage("Administrative password accepted from " + sender);
				this.admins.add(sender);
			} else {
				this.sendMessageResponsibly(sender, "Password rejected");
				this.debugMessage("Administrative password rejected from " + sender + ". They provided: " + password);
			}

		} else {
			for (MessagePlugin mp : this.messagePlugins) {
				mp.onPrivateMessage(this, sender, message);
			}
		}
	}

	@Override
	protected void onServerResponse(final int code, final String response) {
		if (code == ReplyConstants.RPL_WHOISUSER) {
			this.debugMessage("Got WHOIS info back from server.");

			final String parts[] = response.split(" ");

			this.lastWhois = parts[1].toLowerCase() + ":" + parts[2] + "@" + parts[3];
		}
	}

	@Override
	protected void onUnknown(final String line) {
		this.debugMessage("Unknown: " + line);
	}

	public void removeChannelGag(String channel) {
		this.channelGags.remove(channel);
		this.log("Channel gag removed on channel:" + channel);
		this.sendMessage(channel, "Oh, I just woke up. I will talk to this channel again.");
	}

	/**
	 * Attempts to connect to the specified server. Will loop in a seperate
	 * thread until the bot is disconnected. Loops once a seccond.
	 */
	@Override
	public void run() {
		this.connect();

		while (this.isConnected()) {
			try {
				Thread.sleep(1000); // 1 sec
			} catch (final Exception e) {
				System.out.print("Cannot sleep: " + e.toString() + "\n");
				break;
			}
		}

		this.debugMessage("Disconnected from \"" + this.server.getServerName() + "\". \n");
	}

	public void sendMessageResponsibly(String target, String message) {
		if (this.channelGags.contains(target)) {
			Bot.LOG.info("Gagged, wont send - " + target + ": " + message);
		} else {
			Bot.LOG.info("Sending - " + target + ": " + message);
			this.sendMessage(target, message);
		}
	}

	public void sendMessageResponsiblyUser(String channel, String sender, String message) {
		this.sendMessageResponsibly(channel, sender + ": " + message.trim());
	}

	public void sendWhois(final String username) {
		// this.sendRawLineViaQueue("WHOIS " + username);
	}

	public void setOwnerNickname(String newNickname) {
		if (newNickname == null) {
			return;
		}

		this.ownerNickname = newNickname;
	}

	public void setPassword(String password) {
		this.password = password;
	}

	/**
	 * After somebody joins a chanel, setUserModes is called for that user. If
	 * the bot is not an op, it exits. If the user is found in ops.lst, the user
	 * is opped, if not, the user is voiced.
	 */
	private void setUserModes(final String channel, final String sender, final String hostname) {
		if (this.nick.equals(sender)) {
			return;
		}

		if (!this.operator) {
			return;
		}

		try {
			File channelOpsFile = new File("servers/" + this.server.getServerName() + "/channels/" + channel + "/ops.lst");

			if (channelOpsFile.exists()) {
				this.debugMessage("Opened channel ops file: " + channelOpsFile.getAbsolutePath());

				final BufferedReader in = new BufferedReader(new FileReader(channelOpsFile));

				String line = "";

				while ((line = in.readLine()) != null) {
					if (hostname.trim().matches(line)) {
						this.debugMessage("Op'ing: \"" + sender + "\", host: \"" + hostname + "\"\n");

						this.op(channel, sender);
					} else {
						this.debugMessage("Voicing: \"" + sender + "\", host: \"" + hostname + "\"\n");

						this.voice(channel, sender);
					}
				}

				in.close();
			} else {
				this.debugMessage("Channel ops file does not exist: " + channelOpsFile.getAbsolutePath());
				this.debugMessage("Voicing: \"" + sender + "\", host: \"" + hostname + "\"\n");

				this.voice(channel, sender);
			}
		} catch (final Exception e) {
			this.debugMessage("Could not open this channel's op file: " + e.toString());
			return;
		}
	}

	public void setWatchDirectory(File watchDirectory) {
		this.watchDirectory = watchDirectory;
	}

	public void start() {
		this.runner.start();
	}

	public void toggleChannelGag(String channel) {
		if (this.isGagged(channel)) {
			this.removeChannelGag(channel);
		} else {
			this.addChannelGag(channel);
		}
	}

}
