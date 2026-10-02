/**
 * Build one UI session per tool from flat LLM tool run records (newest first preferred).
 * @param {Array} runs
 * @returns {Array}
 */
export function buildLlmToolSessions(runs) {
	const byTool = new Map();
	for (const run of runs || []) {
		const toolId = Number(run.toolId || run.llmToolId || 0);
		if (!toolId) {
			continue;
		}
		let session = byTool.get(toolId);
		if (!session) {
			session = {
				toolId,
				toolName: run.toolName || `Tool #${toolId}`,
				pending: false,
				elapsedLabel: '',
				startedAt: 0,
				history: [],
			};
			byTool.set(toolId, session);
		}
		session.history.push({
			id: run.id,
			status: run.status || 'ok',
			outputContent: run.outputContent || '',
			errorMessage: run.errorMessage || '',
			connectionName: run.connectionName || '',
			model: run.model || '',
			created: run.created || '',
		});
	}
	return Array.from(byTool.values());
}

export function replyFromApiRun(run) {
	return {
		id: run.id,
		status: run.status || 'ok',
		outputContent: run.outputContent || '',
		errorMessage: run.errorMessage || '',
		connectionName: run.connectionName || '',
		model: run.model || '',
		created: run.created || new Date().toISOString(),
	};
}
