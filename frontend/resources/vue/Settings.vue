<template>
	<section class = "settings" title = "Settings">
		<h2>Settings</h2>
		<p>This page allows you to configure all of the cvars available in the app.</p>
	</section>

	<section v-for="category in categories">
		<h2>Category: {{ category.name }}</h2>

		<form>
			<template v-for="cvar in category.cvars">
				<label>{{ cvar.title }}: </label>
				<input :type = "cvar.type" name = "" :placeholder = "cvar.keyName" :value = "cvar.valueString" />
				<span>{{ cvar.description }}</span>
			</template>
		</form>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';

	const categories = ref([]);

	function refreshCvars() {
		categories.value = [];
		window.client.getCvars().then((ret) => {
			categories.value = ret.cvarCategories;
		}).catch((error) => {
			console.error('Error fetching cvars:', error);
		});
	}

	onMounted(async () => {
	    await waitForClient();

		refreshCvars();
	});
</script>

<style scoped>
    form {
		grid-template-columns: 240px 1fr 3fr;
	}

	form {
		align-items: center;
	}

	label {
		justify-self: end;
	}
</style>
