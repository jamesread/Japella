module.exports = {
    "extends": [
        "eslint:recommended",
        "plugin:vue/essential"
    ],
    "parserOptions": {
        "ecmaVersion": 12,
        "sourceType": "module"
    },
    "plugins": [
        "vue"
    ],
    "rules": {
        "vue/multi-word-component-names": "off",
        "vue/require-v-for-key": "off",
        "vue/no-textarea-mustache": "off"
    },
    "ignorePatterns": [
        "dist/**/*",
        "resources/javascript/gen/**/*"
    ]
};
