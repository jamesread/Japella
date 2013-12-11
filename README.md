Quickstart
===
A super simple IRC Bot that supports Java Plugins. Based on the PIRC Library.

This is an example of ~/.japella/config.xml

    <config>
    	<server name = "freenode" address = "irc.freenode.net" port = "6667" />

	<bot name = "japella" serverRef = "freenode" ownerNickname = "myNameHere" password = "mySekritPassword">
		<channel name = "#firstChannel" />
	</bot>
    </config>

Start the bot up, then send it a private message with the password;

    /msg japella !password mySekritPassword

The bot will then well take admin commands. One of the admin commands is telling it to join another channel.

    /msg japella !join #testing

Note about ongoing development
===
This project was built to forful a personal need for a Java based IRC Bot. This bot is feature complete for my needs and I don't have plans to be actively extending or promoting it. It should serve as a good building block for anyone looking to start their own project. 

Patches and forks are totally welcome. The code is GPLv2.

The upstream library is well maintained; http://code.google.com/p/pircbotx/
