<template>
	<section>
		<h2>Calendar</h2>

		<div class = "day-container">
			<div class = "day" v-for = "d in getCalendarDates()" :key = "d">
				{{ d }}
			</div>
		</div>
	</section>
</template>

<script setup>
	import { ref } from 'vue';

	function getCalendarDates() {
		let start = new Date();
		start.setDate(1); // 1st of Month
		start.setDate(start.getDate() - (start.getDay() - 1)) // Monday

		let days = []
		let dayCount = 0;

		for (let week = 0; week < 5; week++) {
			for (let i = 0; i != 7; i++) {
				let day = new Date(start.getTime())
				day.setDate(day.getDate + dayCount);

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
