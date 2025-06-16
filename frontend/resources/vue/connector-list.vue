<template>

	<section class="connector-list">
		<h1>Connectors</h1>
		<form id="post-form" @submit.prevent>
			<label class="grid-wide">Post Mode</label>

			<div class="grid-wide">
				<div v-if="items.length > 0">
					<div v-for="item in items" :key="item.id" class="connector-item">
						<span class="connector-name">{{ item.name }}</span>
						<span class="connector-description">{{ item.description }}</span>
						<label>
							<input type="checkbox" :value="item.id" v-model="selectedConnectors" />
							<span>Select</span>
						</label>
					</div>
				</div>
				<div v-else>
					<p>No posting services available.</p>
				</div>
			</div>

			<label class="grid-wide">Message</label>
			<textarea id="post" rows="8" cols="80" class="grid-wide"></textarea>

			<fieldset>
				<button id="submit" type="submit">Post</button>
			</fieldset>
		</form>
	</section>
</template>


<script setup>
	import { ref, onMounted } from 'vue';
	import RadioGroup from './radio-group.vue';
	import InlineNotification from './inline-notification.vue';
	import { Icon } from '@iconify/vue';
	const clientReady = ref(false);
	const items = ref([]);

</script>
