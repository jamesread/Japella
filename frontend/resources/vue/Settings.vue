<template>
    <section class = "settings" title = "Settings">
        <h2>Settings</h2>
        <p>This page allows you to configure all of the cvars available in the app.</p>

        <InlineNotification message = "Settings are saved immediately when you change them on this page." type = "warning" />
    </section>

    <section v-for="category in categories">
        <h2>Category: {{ category.name }}</h2>

        <form>
            <template v-for="cvar in category.cvars">
                <label>{{ cvar.title }}: </label>
                <!-- Use v-if to handle different input types -->
                    <template v-if = "cvar.type === 'text' || cvar.type === 'password'">
                        <input :type = "cvar.type" name = "" :id = "cvar.keyName" :placeholder = "cvar.keyName" :value = "cvar.valueString" @blur = "setCvar(cvar)" />
                    </template>
                    <template v-else-if = "cvar.type === 'bool'">
                        <input type = "checkbox" :id = "cvar.keyName" :checked = "cvar.valueInt === 1" @blur = "setCvar(cvar)" />
                    </template>
				<span class = "fg1"><div v-html="cvar.description"></div></span>
				<span>
					<a v-if = "cvar.externalUrl" :href = "cvar.externalUrl">Get</a>
				</span>
				<span>
					<a v-if = "cvar.docsUrl" :href = "cvar.docsUrl">Docs</a>
				</span>
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

    function setCvar(cvar) {
        console.log('Blur event for cvar:', cvar);

        let req = {
            keyName: cvar.keyName,
        }

        if (cvar.type === 'text' || cvar.type === 'password') {
            req.valueString = document.getElementById(cvar.keyName).value;
        } else if (cvar.type === 'bool') {
            req.valueInt = document.getElementById(cvar.keyName).checked ? 1 : 0;
        } else if (cvar.type === 'int') {
            req.valueInt = parseFloat(document.getElementById(cvar.keyName).value);
        } else {
            console.warn(`Unsupported cvar type: ${cvar.type}`);
            return;
        }

        window.client.setCvar(req)
        .then(() => {
            console.log(`Cvar ${cvar.keyName} set.`);
        }).catch((error) => {
            console.error(`Error setting cvar ${cvar.keyName}:`, error);
        });
    }

    onMounted(async () => {
        await waitForClient();

        refreshCvars();
    });
</script>

<style scoped>
    form {
        grid-template-columns: 240px 300px 3fr min-content min-content;
    }

    form {
        align-items: center;
    }

    label {
        justify-self: end;
    }
</style>
