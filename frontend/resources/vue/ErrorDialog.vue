<template>
	<dialog ref = "dlg" class = "bad">
		<h2>Error</h2>
		<p>{{ message }}</p>
		<button @click="hide">Close</button>
	</dialog>
</template>

<script setup>
	import { ref } from 'vue';

	const visible = ref(false);
	const message = ref('');
	const dlg = ref(null);

	function showError(msg) {
		message.value = msg;
		visible.value = true;
		dlg.value.showModal();
	}

	function hide() {
		visible.value = false;
		message.value = '';
		dlg.value.close();
	}

	defineExpose({
		showError,
		hide
	});
</script>
