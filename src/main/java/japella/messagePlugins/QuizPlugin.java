package japella.messagePlugins;

import japella.Bot;
import japella.MessagePlugin;
import japella.configuration.PropertiesFileCollection;

import java.util.Collections;
import java.util.HashMap;
import java.util.Vector;

import org.apache.commons.configuration.ConfigurationException;
import org.apache.commons.configuration.PropertiesConfiguration;

public class QuizPlugin extends MessagePlugin {
	class QuizQuestion {
		String question;
		String answer;
		String createdBy;
	}

	private QuizQuestion currentQuestion = null;

	private final Vector<QuizQuestion> activeQuestions = new Vector<QuizPlugin.QuizQuestion>();

	public Vector<QuizQuestion> questions = new Vector<QuizQuestion>();

	private final HashMap<String, Integer> scoreboard = new HashMap<String, Integer>();

	private PropertiesConfiguration properties;

	public QuizPlugin() {
		try {
			this.properties = PropertiesFileCollection.get(this);

			if (!this.properties.containsKey("questionCount")) {
				this.properties.setProperty("questionCount", 0);
			}
		} catch (Exception e) {
			e.printStackTrace();
		}

		this.loadConfig();
	}

	@CommandMessage(keyword = "!quizaddquestion")
	public void addQuizQuestion(Message message2) {
		Bot bot = message2.bot;
		String message = message2.originalMessage;

		String[] messageParts = message.split("=", 2);

		if (messageParts.length != 2) {
			message2.reply("To add a quiz question, use the syntax: <question>=<answer>");
		} else {
			QuizQuestion newQuestion = new QuizQuestion();
			newQuestion.question = messageParts[0].replace("!quizaddquestion ", "").trim();
			newQuestion.answer = messageParts[1].trim();
			newQuestion.createdBy = message2.sender;

			this.questions.add(newQuestion);

			message2.reply("Question accepted.");
		}
	}

	private void askNextQuestion(Bot bot, String channel) {
		this.currentQuestion = this.activeQuestions.lastElement();

		if (this.currentQuestion != null) {
			this.activeQuestions.remove(this.currentQuestion);
			bot.sendMessageResponsibly(channel, "Quiz question: " + this.currentQuestion.question);
		}
	}

	public void clearQuestions() {
		this.questions.clear();
	}

	@CommandMessage(keyword = "!quizload")
	public void commandLoad(Message message) {
		if (this.isQuizRunning()) {
			message.reply("A quiz is running, I cannot load the config file right now.");
		} else {
			this.loadConfig();

			message.reply("The quiz database now contains " + this.questions.size() + " questions.");
		}
	}

	@CommandMessage(keyword = "!quizsave")
	public void commandSave(Message message) {
		this.saveConfig();
	}

	public Vector<QuizQuestion> getActiveQuestions() {
		return this.activeQuestions;
	}

	public QuizQuestion getCurrentQuestion() {
		return this.currentQuestion;
	}

	private String getHighestPlayer() {
		int currentScore = 0;
		String currentPlayer = "";

		for (String player : this.scoreboard.keySet()) {
			if (this.scoreboard.get(player) > currentScore) {
				currentPlayer = player;
			}
		}

		return currentPlayer;
	}

	@Override
	public String getName() {
		return this.getClass().getSimpleName();
	}

	public Vector<QuizQuestion> getQuestions() {
		return this.questions;
	}

	public HashMap<String, Integer> getScoreboard() {
		return this.scoreboard;
	}

	private boolean isQuizRunning() {
		return !this.activeQuestions.isEmpty();
	}

	public void loadConfig() {
		try {
			this.properties.refresh();
		} catch (ConfigurationException e) {
			e.printStackTrace();
			return;
		}

		int questionCount = this.properties.getInt("questionCount");

		this.questions.clear();

		for (int i = 0; i <= questionCount; i++) {
			QuizQuestion question = new QuizQuestion();

			question.question = this.properties.getString("questions." + i + ".question");
			question.answer = this.properties.getString("questions." + i + ".answer");
			question.createdBy = this.properties.getString("questions." + i + ".createdBy");

			this.questions.add(question);
		}
	}

	@CommandMessage(keyword = "!quizstop")
	public void onAbortQuiz(Message message) {
		this.scoreboard.clear();
		this.activeQuestions.clear();

		message.reply("Quiz stopped.");
	}

	@CommandMessage(keyword = "!guess")
	public void onGuess(Message message) {
		String answer = message.originalMessage.replace("!guess", "").trim();
		message.parser.toString().replace("!guess", "").trim();

		if (this.currentQuestion == null) {
			message.reply("There is no active quiz. To start a new quiz, use !quizstart.");
			return;
		}

		if (answer.equalsIgnoreCase(this.currentQuestion.answer)) {
			message.reply("Correct answer, " + message.sender + "!");

			if (!this.scoreboard.containsKey(message.sender)) {
				this.scoreboard.put(message.sender, 0);
			}

			this.scoreboard.put(message.sender, this.scoreboard.get(message.sender) + 1);

			if (this.activeQuestions.isEmpty()) {
				String winningPlayer = this.getHighestPlayer();
				int winningScore = this.scoreboard.get(winningPlayer);
				this.currentQuestion = null;

				message.reply("The quiz is over. The winner is: " + winningPlayer + " with " + winningScore + " point(s)");
			} else {
				this.askNextQuestion(message.bot, message.channel);
			}
		} else {
			message.reply("Wrong answer!");
			message.bot.log(message.sender + " guessed " + answer + ", but the answer was " + this.currentQuestion.answer);
		}
	}

	@CommandMessage(keyword = "!quizstart")
	public void onNewQuiz(Message message) {
		if (this.isQuizRunning()) {
			message.reply("I won't start a new quiz, because one is already running.");
			return;
		}

		if (this.questions.isEmpty()) {
			message.reply("I won't start a new quiz, because there are 0 questions in the datatabase. PM me with !quizadd");
			return;
		}

		if (!message.parser.hasParam(1)) {
			message.reply("You need to tell me how many questions you want in the quiz!");
			return;
		}

		int questionCount;

		try {
			questionCount = message.parser.getInt(1);
		} catch (NumberFormatException e) {
			questionCount = 3;
		}

		questionCount = Math.min(questionCount, this.questions.size());

		this.activeQuestions.addAll(this.questions);
		Collections.shuffle(this.activeQuestions);
		this.activeQuestions.setSize(questionCount);

		this.scoreboard.clear();

		message.reply("A new quiz has been started with " + this.activeQuestions.size() + " questions - type \"!guess <answer string>\" to play!");
		this.askNextQuestion(message.bot, message.channel);
	}

	@CommandMessage(keyword = "!quizquestioncount")
	public void onQuizQCount(Message message) {
		message.reply("The quiz has " + this.questions.size() + " question(s) in the database.");
	}

	@Override
	public void saveConfig() {
		try {
			int i = 0;
			for (QuizQuestion question : this.questions) {
				i++;
				this.properties.setProperty("questions." + i + ".question", question.question);
				this.properties.setProperty("questions." + i + ".answer", question.answer);
				this.properties.setProperty("questions." + i + ".createdBy", question.createdBy);
			}

			this.properties.setProperty("questionCount", i);

			this.properties.save();
		} catch (ConfigurationException e) {
			e.printStackTrace();
		}
	}
}
