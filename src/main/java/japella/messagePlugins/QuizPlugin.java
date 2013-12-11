package japella.messagePlugins;

import java.util.Collections;
import java.util.HashMap;
import java.util.Vector;


import japella.Bot;
import japella.MessagePlugin;

public class QuizPlugin extends MessagePlugin {
	static class QuizQuestion {
		String question;
		String answer;
	}
	  
	private QuizQuestion currentQuestion = null;
	private Vector<QuizQuestion> activeQuestions = new Vector<QuizPlugin.QuizQuestion>();
	public Vector<QuizQuestion> questions = new Vector<QuizQuestion>();
	private HashMap<String, Integer> scoreboard = new HashMap<String, Integer>();
	  
	public void addMessage(String m) { 
	}

	public void onMessage(Bot bot, String channel, String sender, String login, String hostname, String message) {
		if (message.contains("!newquiz")) {
			System.out.println("new quiz");
			
			if (!activeQuestions.isEmpty()) {
				bot.sendMessageResponsibly(channel, sender + ": I won't start a new quiz, because one is already running.");
				return;    
			}     
 			
			if (questions.isEmpty()) { 
				bot.sendMessageResponsibly(channel, sender + ": I won't start a new quiz, because there are 0 questions in the datatabase. PM me with !quizadd");
				return;
			}
			
			int questionCount;
			
			try {
				questionCount = Integer.parseInt(message.replace("!newquiz", "").trim());
			} catch (NumberFormatException e) {
				questionCount = 3;
			}
			
			questionCount = Math.min(questionCount, questions.size());
			 
			activeQuestions.addAll(questions);  
			Collections.shuffle(activeQuestions); 
			activeQuestions.setSize(questionCount); 
			 
			scoreboard.clear(); 
 			     
			bot.sendMessageResponsibly(channel, "A new quiz has been started with " + activeQuestions.size() + " questions - type \"!guess <answer string>\" to play!");
			askNextQuestion(bot, channel);  
		} else if (message.contains("!guess")) {
			String answer = message.replace("!guess", "").trim(); 
			
			if (currentQuestion == null) {
				bot.sendMessageResponsiblyUser(channel, sender, "There is no active quiz. To start a new quiz, use !newquiz.");
				return; 
			}
			
			if (answer.equalsIgnoreCase(currentQuestion.answer)) {
				bot.sendMessageResponsibly(channel, "Correct answer, " + sender + "!");
				
				if (!scoreboard.containsKey(sender)) {
					scoreboard.put(sender, 0);
				}
				 
				scoreboard.put(sender, scoreboard.get(sender) + 1);
				
				if (activeQuestions.isEmpty()) { 
					String winningPlayer = this.getHighestPlayer();
					int winningScore = scoreboard.get(winningPlayer);
					
					this.currentQuestion = null;
 					
					bot.sendMessageResponsibly(channel, "The quiz is over. The winner is: " + winningPlayer + " with " + winningScore + " point(s)");
				} else {
					askNextQuestion(bot, channel);
				}
			} else {   
				bot.sendMessageResponsiblyUser(channel, sender, "Wrong answer!"); 
				bot.log(sender + " guessed " + answer + ", but the answer was " + currentQuestion.answer); 
			}
		} else if (message.contains("!quizqcount")) {  
			bot.sendMessageResponsiblyUser(channel, sender, "The quiz has " + questions.size() + " in the database.");
		} else if (message.contains("!abortquiz" )) {
			scoreboard.clear();
			activeQuestions.clear();
			 
			bot.sendMessageResponsibly(channel, "Quiz aborted.");
		}
	}
	
	private void askNextQuestion(Bot bot, String channel) {
		this.currentQuestion = activeQuestions.lastElement();
		
		if (currentQuestion != null) {
			activeQuestions.remove(currentQuestion); 
			bot.sendMessageResponsibly(channel, "Quiz question: " + currentQuestion.question);
		}
	}
	
	private String getHighestPlayer() {
		int currentScore = 0;
		String currentPlayer = ""; 
		
		for (String player : scoreboard.keySet()) {
			if (scoreboard.get(player) > currentScore) { 
				currentPlayer = player;
			}
		}
		
		return currentPlayer;
	}

	public String getName() {
		return this.getClass().getSimpleName();
	}

	public void onTimerTick(Bot bot, String channel) {
	}

	public void onPrivateMessage(Bot bot, String sender, String message) {
		if (message.contains("!quizadd")) {
			String[] messageParts = message.split("=", 2);
			
			if (messageParts.length != 2) { 
				bot.sendMessageResponsibly(sender, "To add a quiz question, use the syntax: <question>=<answer>");
			} else { 
				QuizQuestion newQuestion = new QuizQuestion(); 
				newQuestion.question = messageParts[0].replace("!quizadd ", "").trim();   
				newQuestion.answer = messageParts[1].trim();  
				
				questions.add(newQuestion);   
				
				bot.sendMessageResponsibly(sender, "Question accepted."); 
			} 
		}
	} 
}
