<template>
	<section>
		<h2>Calendar</h2>

		<div class = "day-container">
			<div class = "day" v-for = "d in getCalendarDates()" :key = "d">
				{{ formatDate(d) }}
			</div>
		</div>
	</section>
</template>

<script setup>
	import { ref } from 'vue';


	function ordinalSuffix(n) {
		let suffix = ['th', 'st', 'nd', 'rd']
		let value = n % 100
		return n + (suffix[(value - 20) % 10] || suffix[value] || suffix[0])
	}

	function formatDate(d) {
		const day = d.getDate()
		const month = d.toLocaleString('default', { month: 'long' })

		return `${ordinalSuffix(day)} ${month}`
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
		gap: 4px;
	}

	.day {
		border: 1px solid #efefef;
	}
</style>
