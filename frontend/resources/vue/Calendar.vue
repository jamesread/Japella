<template>
	<Section
		title="Calendar"
		subtitle="This page shows a calendar for the current month."
		classes="calendar"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refreshEvents" :disabled="!clientReady" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="errorMessage">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>
		<div v-else>
			<Calendar
				:events="events"
				:loading="loading"
				:error="errorMessage"
				@date-click="handleDateClick"
				@event-click="handleEventClick"
				@month-change="handleMonthChange"
			/>
		</div>
	</Section>
</template>

<script setup>
    import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Calendar from 'picocrank/vue/components/Calendar.vue';
	import Notification from '../javascript/notification.js';

	const clientReady = ref(false);
	const loading = ref(false);
	const errorMessage = ref('');
	const events = ref([]);
    const currentMonth = ref(new Date().getMonth());
    const currentYear = ref(new Date().getFullYear());

	function handleDateClick(date) {
		const formattedDate = date.toLocaleDateString();
		const notification = new Notification('good', 'Calendar Date', `You clicked on ${formattedDate}`);
		notification.show();
		console.log('Date clicked:', formattedDate);
	}

	function handleEventClick(event) {
		const notification = new Notification('good', 'Calendar Event', `You clicked on event: ${event.title}`);
		notification.show();
		console.log('Event clicked:', event);
	}

    function handleMonthChange(month, year) {
        console.log('Month changed to:', month, year);
        currentMonth.value = month;
        currentYear.value = year;
        refreshEvents();
    }

    async function refreshEvents() {
        if (!window.client) {
            errorMessage.value = "Client is not ready.";
            return;
        }

        loading.value = true;
        errorMessage.value = '';

        try {
            // Fetch timeline posts and map them to calendar events using the created timestamp
            const ret = await window.client.getTimeline();
            const posts = (ret && ret.posts) ? ret.posts : [];

            const mapped = posts
                .map((p) => {
                    // Parse created date safely
                    let d = new Date(p.created);
                    if (Number.isNaN(d.getTime()) && typeof p.created === 'string') {
                        // Try common non-ISO format fallback (e.g. "YYYY-MM-DD HH:mm:ss")
                        d = new Date(p.created.replace(' ', 'T'));
                    }
                    if (Number.isNaN(d.getTime()) && typeof p.created === 'number') {
                        d = new Date(p.created);
                    }
                    if (Number.isNaN(d.getTime())) {
                        return null; // skip if still invalid
                    }

                    const titleFromIdentity = p.socialAccountIdentity && p.socialAccountIdentity.trim().length > 0
                        ? p.socialAccountIdentity
                        : 'Post';
                    const contentPreview = (p.content || '').trim();
                    const preview = contentPreview.length > 80
                        ? contentPreview.slice(0, 77) + '...'
                        : contentPreview;

                    return {
                        id: p.id,
                        title: preview || titleFromIdentity,
                        date: d,
                        startDate: d,
                        endDate: d,
                    };
                })
                .filter((e) => e !== null);

            events.value = mapped;
        } catch (error) {
            errorMessage.value = `Failed to fetch events: ${error.message}`;
            console.error('Error fetching events:', error);
        } finally {
            loading.value = false;
        }
    }

    onMounted(async () => {
        await waitForClient();
        clientReady.value = true;
        const now = new Date();
        currentMonth.value = now.getMonth();
        currentYear.value = now.getFullYear();
        refreshEvents();
    });
</script>

<style scoped>
	/* Custom calendar styling can be added here if needed */
</style>
