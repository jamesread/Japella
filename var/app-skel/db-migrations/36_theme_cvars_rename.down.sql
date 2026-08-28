UPDATE `cvars`
SET
  `key_name` = 'enable_theme_color_scheme_switcher',
  `title` = 'Theme color scheme switcher',
  `description` = 'When enabled, logged-in users see the light/dark/auto theme control in the header.',
  `category` = 'Theme'
WHERE `key_name` = 'theme_color_scheme_switcher_enabled';
