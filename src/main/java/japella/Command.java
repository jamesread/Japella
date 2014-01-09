package japella;

import java.lang.reflect.Type;

public class Command {
	private final String[] parts;

	private final String originalMessage;

	public Command(String command) {
		this.originalMessage = command;
		this.parts = command.trim().split(" ");
	}

	public int getInt(int position) {
		String i = this.parts[position];

		if (!this.isInt(position)) {
			return 0;
		} else {
			return Integer.parseInt(i);
		}
	}

	public String getOriginalMessage() {
		return this.originalMessage;
	}

	public String getString(int position) {
		return this.parts[position];
	}

	public boolean hasParam(int position) {
		if (position >= this.parts.length) {
			return false;
		} else {
			return true;
		}
	}

	public boolean isInt(int position) {
		if (!this.hasParam(position)) {
			return false;
		} else {
			String i = this.parts[position];

			try {
				Integer.parseInt(i);
				return true;
			} catch (Exception e) {
				return false;
			}
		}
	}

	private boolean isString(int position) {
		return this.hasParam(position);
	}

	public boolean matches(Type... types) throws Exception {
		for (int i = 0; i < types.length; i++) {
			Type t = types[i];

			if (!this.hasParam(i)) {
				throw new Exception("Expected a parameter at position:" + i);
			} else {
				if (t == String.class) {
					if (this.isString(i)) {
						continue;
					} else {
						throw new Exception("Expected parameter " + i + " to be String.");
					}
				} else if (t == Integer.class) {
					if (this.isInt(i)) {
						continue;
					} else {
						throw new Exception("Expected parameter " + i + " to be int");
					}
				} else {
					throw new Exception("Parameter in position " + i + " was not a string or an int!");
				}
			}
		}

		return true;
	}

	public boolean supportsKeyword(String keyword) {
		return this.getString(0).equalsIgnoreCase(keyword);
	}
}
