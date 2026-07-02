<template>
	<div v-if="issues?.length" class="connector-issues">
		<h3 v-if="showTitle">Configuration issues</h3>
		<p
			v-for="(issue, index) in issues"
			:key="issueKey(issue, index)"
			class="inline-notification error small"
		>
			{{ issue.message }}
			<button
				v-if="issue.fixAction === registerClientAction && !registering"
				type="button"
				class="issue-fix-button"
				@click="$emit('register-client')"
			>
				{{ issue.fixLabel || 'Register OAuth application' }}
			</button>
			<button
				v-else-if="issue.fixAction === registerClientAction && registering"
				type="button"
				class="issue-fix-button"
				disabled
			>
				Registering…
			</button>
			<router-link
				v-else-if="issueFixRoute(issue)"
				:to="issueFixRoute(issue)"
				class="issue-fix-link"
			>
				{{ issue.fixLabel || 'Fix in Settings' }}
			</router-link>
		</p>
	</div>
</template>

<script setup>
	import { connectorIssueFixRoute, CONNECTOR_FIX_ACTION_REGISTER_CLIENT } from '../javascript/util';

	defineProps({
		issues: {
			type: Array,
			default: () => [],
		},
		showTitle: {
			type: Boolean,
			default: true,
		},
		registering: {
			type: Boolean,
			default: false,
		},
	});

	defineEmits(['register-client']);

	const registerClientAction = CONNECTOR_FIX_ACTION_REGISTER_CLIENT;

	function issueFixRoute(issue) {
		return connectorIssueFixRoute(issue);
	}

	function issueKey(issue, index) {
		return `${issue.message}-${issue.fixPath || ''}-${issue.fixHash || ''}-${issue.fixAction || ''}-${index}`;
	}
</script>

<style scoped>
	.connector-issues h3 {
		margin: 0 0 0.5rem;
		font-size: 1rem;
	}

	.connector-issues {
		margin-top: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.issue-fix-link,
	.issue-fix-button {
		display: inline-block;
		margin-left: 0.5rem;
		font-weight: 600;
		white-space: nowrap;
	}

	.issue-fix-button {
		border: 0;
		background: transparent;
		color: inherit;
		text-decoration: underline;
		cursor: pointer;
		padding: 0;
		font: inherit;
	}

	.small {
		font-size: 0.85em;
	}
</style>
