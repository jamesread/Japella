<template>
	<Teleport to="body">
		<div
			class="notification-popup-stack"
			aria-live="polite"
			aria-relevant="additions"
		>
			<TransitionGroup name="notification-popup">
				<NotificationPopup
					v-for="popup in notificationPopups"
					:key="popup.id"
					:popup="popup"
					@dismiss="dismissNotificationPopup"
				/>
			</TransitionGroup>
		</div>
	</Teleport>
</template>

<script setup>
import { onUnmounted } from 'vue';
import NotificationPopup from 'picocrank/vue/components/NotificationPopup.vue';
import {
	notificationPopups,
	dismissNotificationPopup,
	dismissAllNotificationPopups,
	showNotificationPopup,
	clearNotificationPopupTimers,
} from '../javascript/notifications.js';

onUnmounted(() => {
	clearNotificationPopupTimers();
});

defineExpose({
	popups: notificationPopups,
	show: showNotificationPopup,
	dismiss: dismissNotificationPopup,
	dismissAll: dismissAllNotificationPopups,
});
</script>

<style scoped>
.notification-popup-stack {
	position: fixed;
	right: 1rem;
	bottom: 1rem;
	z-index: 1000;
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
	pointer-events: none;
}

.notification-popup-stack :deep(.notification-popup) {
	pointer-events: auto;
}

.notification-popup-enter-active,
.notification-popup-leave-active {
	transition: opacity 0.4s ease, transform 0.4s ease;
}

.notification-popup-enter-from,
.notification-popup-leave-to {
	opacity: 0;
	transform: translateX(1rem);
}

.notification-popup-move {
	transition: transform 0.3s ease;
}
</style>
