package japella.messagePlugins;

import japella.Bot;
import japella.MessagePlugin;

import java.util.Collections;
import java.util.HashMap;
import java.util.Vector;

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

	@CommandMessage(keyword = "!addquizquestion")
	public void addQuizQuestion(Message message2) {
		Bot bot = message2.bot;
		String message = message2.originalMessage;

		String[] messageParts = message.split("=", 2);

		if (messageParts.length != 2) {
			message2.reply("To add a quiz question, use the syntax: <question>=<answer>");
		} else {
			QuizQuestion newQuestion = new QuizQuestion();
			newQuestion.question = messageParts[0].replace("!quizadd ", "").trim();
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

	@CommandMessage(keyword = "!abortquiz")
	public void onAbortQuiz(Message message) {
		this.scoreboard.clear();
		this.activeQuestions.clear();

		message.reply("Quiz aborted.");
	}

	@Override
	public void onChannelMessage(Bot bot, String channel, String sender, String login, String hostname, String message) {
		if (message.contains("!guess")) {
			String answer = message.replace("!guess", "").trim();

			if (this.currentQuestion == null) {
				bot.sendMessageResponsiblyUser(channel, sender, "There is no active quiz. To start a new quiz, use !newquiz.");
				return;
			}

			if (answer.equalsIgnoreCase(this.currentQuestion.answer)) {
				bot.sendMessageResponsibly(channel, "Correct answer, " + sender + "!");

				if (!this.scoreboard.containsKey(sender)) {
					this.scoreboard.put(sender, 0);
				}

				this.scoreboard.put(sender, this.scoreboard.get(sender) + 1);

				if (this.activeQuestions.isEmpty()) {
					String winningPlayer = this.getHighestPlayer();
					int winningScore = this.scoreboard.get(winningPlayer);
					this.currentQuestion = null;

					bot.sendMessageResponsibly(channel, "The quiz is over. The winner is: " + winningPlayer + " with " + winningScore + " point(s)");
				} else {
					this.askNextQuestion(bot, channel);
				}
			} else {
				bot.sendMessageResponsiblyUser(channel, sender, "Wrong answer!");
				bot.log(sender + " guessed " + answer + ", but the answer was " + this.currentQuestion.answer);
			}
		} else if (message.contains("!quizqcount")) {
			bot.sendMessageResponsiblyUser(channel, sender, "The quiz has " + this.questions.size() + " in the database.");
		}
	}

	@CommandMessage(keyword = "!newquiz")
	public void onNewQuiz(Message message) {
		Bot bot = message.bot;
		String channel = message.channel;
		String sender = message.sender;

		if (!this.activeQuestions.isEmpty()) {
			bot.sendMessageResponsibly(channel, sender + ": I won't start a new quiz, because one is already running.");
			return;
		}

		if (this.questions.isEmpty()) {
			bot.sendMessageResponsibly(channel, sender + ": I won't start a new quiz, because there are 0 questions in the datatabase. PM me with !quizadd");
			return;
		}

		int questionCount;

		try {
			questionCount = message.command.getInt(1);
		} catch (NumberFormatException e) {
			questionCount = 3;
		}

		questionCount = Math.min(questionCount, this.questions.size());

		this.activeQuestions.addAll(this.questions);
		Collections.shuffle(this.activeQuestions);
		this.activeQuestions.setSize(questionCount);

		this.scoreboard.clear();

		bot.sendMessageResponsibly(channel, "A new quiz has been started with " + this.activeQuestions.size() + " questions - type \"!guess <answer string>\" to play!");
		this.askNextQuestion(bot, channel);
	}
}
