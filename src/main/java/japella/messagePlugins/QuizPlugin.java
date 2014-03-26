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

	@CommandMessage(keyword = "!guess")
	public void onGuess(Message message) {
		String answer = message.originalMessage.replace("!guess", "").trim();
		message.parser.toString().replace("!guess", "").trim();

		if (this.currentQuestion == null) {
			message.reply("There is no active quiz. To start a new quiz, use !newquiz.");
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

	@CommandMessage(keyword = "!newquiz")
	public void onNewQuiz(Message message) {
		if (!this.activeQuestions.isEmpty()) {
			message.reply("I won't start a new quiz, because one is already running.");
			return;
		}

		if (this.questions.isEmpty()) {
			message.reply("I won't start a new quiz, because there are 0 questions in the datatabase. PM me with !quizadd");
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
		message.reply("The quiz has " + this.questions.size() + " in the database.");
	}
}
