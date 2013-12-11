A super simple IRC Bot that supports Java Plugins. Based on the PIRC Library.

This is an example of ~/.japella/config.xml

    <config>
    	<server name = "freenode" address = "irc.freenode.net" port = "6667" />

	<bot name = "japella" serverRef = "freenode" ownerNickname = "myNameHere" password = "mySekritPassword" />
    </config>

Start the bot up, then send it a private message with the password;

    /msg japella !password mySekritPassword

The bot will then respond to you. Enjoy.
