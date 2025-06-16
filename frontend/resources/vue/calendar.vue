<template>
	<section class = "calendar">
		<h2>Calendar</h2>

		<p>This page shows a calendar for the current month.</p>

		<div class = "day-container">
			<div class = "day" v-for = "d in getCalendarDates()" :key = "d">
				<button @click = "showNotification(d)" class = "neutral">
					<small>{{ formatDate(d) }}</small>
				</button>
			</div>
		</div>
	</section>
</template>

<script setup>
	import { ref } from 'vue';
	import Notification from '../javascript/notification.js';

	function showNotification(date) {
		let formattedDate = formatDate(date);
		let notification = new Notification('good', ',Calendar Date', `You clicked on ${formattedDate}`);

		notification.show();
		console.log('Notification shown for date:', formattedDate);
	}

	function ordinalSuffix(n) {
		let suffix = ['th', 'st', 'nd', 'rd']
		let value = n % 100
		return n + (suffix[(value - 20) % 10] || suffix[value] || suffix[0])
	}

	function formatDate(d) {
		const day = d.getDate()
		const dayName = d.toLocaleString('default', { weekday: 'short' })
		const month = d.toLocaleString('default', { month: 'long' })

		return `${dayName} ${ordinalSuffix(day)}`
	}

	function getCalendarDates() {
		let start = new Date();
		start.setDate(1); // 1st of Month
		start.setDate(start.getDate() - (start.getDay() - 1)) // Monday

		let days = []
		let dayCount = 0;

		for (let week = 0; week < 5; week++) {
			for (let i = 0; i != 7; i++) {
				let day = new Date(start.getTime())
				day.setDate(day.getDate() + dayCount);

				days.push(day)
				dayCount++
			}
		}



		return days
	}
</script>

<style lang = "css" scoped>
	.day-container {
		display: grid;
		grid-template-rows: repeat(5, auto);
		grid-template-columns: repeat(7, auto);
		gap: 0;
		/** grid border collapse */
		border-collapse: collapse;
		box-sizing: border-box;


	}

	.day {
		outline: 1px solid #666;
		padding: 8px;
		min-height: 120px;
	}
</style>
