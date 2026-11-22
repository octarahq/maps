package com.octara.maps.ui.theme

import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.graphics.luminance

val PrimaryLight = Color(0xFF0061A4)
val OnPrimaryLight = Color(0xFFFFFFFF)
val PrimaryContainerLight = Color(0xFFD1E4FF)
val OnPrimaryContainerLight = Color(0xFF001D36)
val SecondaryLight = Color(0xFF535F70)
val OnSecondaryLight = Color(0xFFFFFFFF)
val SecondaryContainerLight = Color(0xFFD7E3F7)
val OnSecondaryContainerLight = Color(0xFF101C2B)
val TertiaryLight = Color(0xFF6B5778)
val OnTertiaryLight = Color(0xFFFFFFFF)
val TertiaryContainerLight = Color(0xFFF2DAFF)
val OnTertiaryContainerLight = Color(0xFF251431)
val ErrorLight = Color(0xFFBA1A1A)
val OnErrorLight = Color(0xFFFFFFFF)
val ErrorContainerLight = Color(0xFFFFDAD6)
val OnErrorContainerLight = Color(0xFF410002)
val BackgroundLight = Color(0xFFFDFCFF)
val OnBackgroundLight = Color(0xFF1A1C1E)
val SurfaceLight = Color(0xFFFDFCFF)
val OnSurfaceLight = Color(0xFF1A1C1E)
val SurfaceVariantLight = Color(0xFFDFE2EB)
val OnSurfaceVariantLight = Color(0xFF43474E)
val OutlineLight = Color(0xFF73777F)
val OutlineVariantLight = Color(0xFFC3C7CF)

val PrimaryDark = Color(0xFF9ECAFF)
val OnPrimaryDark = Color(0xFF003258)
val PrimaryContainerDark = Color(0xFF00497D)
val OnPrimaryContainerDark = Color(0xFFD1E4FF)
val SecondaryDark = Color(0xFFBBC7DB)
val OnSecondaryDark = Color(0xFF253140)
val SecondaryContainerDark = Color(0xFF3B4858)
val OnSecondaryContainerDark = Color(0xFFD7E3F7)
val TertiaryDark = Color(0xFFD6BEE4)
val OnTertiaryDark = Color(0xFF3B2948)
val TertiaryContainerDark = Color(0xFF523F5F)
val OnTertiaryContainerDark = Color(0xFFF2DAFF)
val ErrorDark = Color(0xFFFFB4AB)
val OnErrorDark = Color(0xFF690005)
val ErrorContainerDark = Color(0xFF93000A)
val OnErrorContainerDark = Color(0xFFFFDAD6)
val BackgroundDark = Color(0xFF111318)
val OnBackgroundDark = Color(0xFFE2E2E6)
val SurfaceDark = Color(0xFF111318)
val OnSurfaceDark = Color(0xFFE2E2E6)
val SurfaceVariantDark = Color(0xFF43474E)
val OnSurfaceVariantDark = Color(0xFFC3C7CF)
val OutlineDark = Color(0xFF8D9199)
val OutlineVariantDark = Color(0xFF43474E)

@Immutable
data class AppSemanticColors(
    val isDark: Boolean,
    val delayedText: Color,
    val delayedContainer: Color,
    val earlyText: Color,
    val warningContainer: Color,
    val warningIcon: Color,
    val warningText: Color,
    val alertContainer: Color,
    val alertContent: Color,
    val blockedCross: Color,
    val pastBlendTarget: Color
)

val LightAppSemanticColors = AppSemanticColors(
    isDark = false,
    delayedText = Color(0xFFD84315),
    delayedContainer = Color(0xFFFFF3E0),
    earlyText = Color(0xFF2E7D32),
    warningContainer = Color(0xFFFFF8E1),
    warningIcon = Color(0xFFEF6C00),
    warningText = Color(0xFFE65100),
    alertContainer = Color(0xFFFFF3E0),
    alertContent = Color(0xFFE65100),
    blockedCross = Color(0xFFE53935),
    pastBlendTarget = Color(0xFF9E9E9E)
)

val DarkAppSemanticColors = AppSemanticColors(
    isDark = true,
    delayedText = Color(0xFFFF8A65),
    delayedContainer = Color(0xFF4A2511),
    earlyText = Color(0xFF81C784),
    warningContainer = Color(0xFF3E2D12),
    warningIcon = Color(0xFFFFB74D),
    warningText = Color(0xFFFFCC80),
    alertContainer = Color(0xFF4A2511),
    alertContent = Color(0xFFFFAB91),
    blockedCross = Color(0xFFEF5350),
    pastBlendTarget = Color(0xFF55555E)
)

val LocalAppColors = staticCompositionLocalOf { LightAppSemanticColors }

fun Color.ensureContrastForTheme(isDark: Boolean): Color {
    val lum = luminance()
    return if (isDark) {
        if (lum < 0.16f) {
            val fraction = ((0.16f - lum) / 0.16f * 0.45f + 0.15f).coerceIn(0f, 0.55f)
            lerp(this, Color.White, fraction)
        } else {
            this
        }
    } else {
        if (lum > 0.52f) {
            val fraction = ((lum - 0.52f) / 0.48f * 0.35f + 0.18f).coerceIn(0f, 0.48f)
            lerp(this, Color.Black, fraction)
        } else {
            this
        }
    }
}
