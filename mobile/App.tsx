import { StatusBar } from "expo-status-bar";
import { Image, ScrollView, StyleSheet, Text, View } from "react-native";

const colors = {
  background: "#0D100C",
  surface: "#171D15",
  border: "#303A2A",
  text: "#F1F2E9",
  muted: "#ABB3A2",
  accent: "#C5F477",
  quiet: "#252D22",
};

export default function App() {
  return (
    <ScrollView style={styles.safeArea}>
      <StatusBar style="light" />
      <ScrollView contentContainerStyle={styles.page}>
        <View style={styles.header}>
          <Image
            source={require("./assets/machineward-avatar.png")}
            style={styles.mark}
            accessibilityElementsHidden
          />
          <Text style={styles.wordmark}>machineward</Text>
          <Text style={styles.version}>V0 PREVIEW</Text>
        </View>

        <View style={styles.intro}>
          <Text style={styles.eyebrow}>YOUR MACHINES</Text>
          <Text style={styles.title}>Know what your computer is doing.</Text>
          <Text style={styles.body}>
            Pair a computer to see its health and activity here.
          </Text>
        </View>

        <View style={styles.emptyCard}>
          <View style={styles.statusRow}>
            <View style={styles.statusDot} />
            <Text style={styles.statusText}>No computer connected</Text>
          </View>
          <View style={styles.rule} />
          <Text style={styles.cardTitle}>Start with your Linux PC</Text>
          <Text style={styles.cardBody}>
            Machineward is in early development. Secure pairing is the next part
            of the build.
          </Text>
          <View style={styles.disabledButton}>
            <Text style={styles.disabledButtonText}>Pair a computer</Text>
            <Text style={styles.buttonArrow}>↗</Text>
          </View>
          <Text style={styles.note}>Pairing is not available yet</Text>
        </View>

        <View style={styles.footer}>
          <Text style={styles.footerText}>
            Linux first · Your machine, your data, your control.
          </Text>
          <Text style={styles.footerVersion}>Built in public by EmannCode</Text>
        </View>
      </ScrollView>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  page: {
    flexGrow: 1,
    paddingHorizontal: 24,
    paddingTop: 12,
    paddingBottom: 32,
  },
  header: {
    minHeight: 44,
    flexDirection: "row",
    alignItems: "center",
  },
  mark: {
    width: 28,
    height: 28,
    marginRight: 10,
  },
  wordmark: {
    color: colors.text,
    fontSize: 16,
    fontWeight: "700",
    letterSpacing: -0.4,
  },
  version: {
    marginLeft: "auto",
    color: colors.muted,
    fontSize: 9,
    letterSpacing: 1.1,
  },
  intro: {
    marginTop: 54,
    marginBottom: 26,
  },
  eyebrow: {
    color: colors.accent,
    fontSize: 10,
    letterSpacing: 1.8,
    marginBottom: 14,
  },
  title: {
    maxWidth: 340,
    color: colors.text,
    fontSize: 34,
    lineHeight: 40,
    fontWeight: "600",
    letterSpacing: -1.2,
  },
  body: {
    maxWidth: 330,
    marginTop: 12,
    color: colors.muted,
    fontSize: 15,
    lineHeight: 23,
  },
  emptyCard: {
    padding: 20,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface,
  },
  statusRow: {
    minHeight: 24,
    flexDirection: "row",
    alignItems: "center",
  },
  statusDot: {
    width: 7,
    height: 7,
    borderRadius: 4,
    marginRight: 9,
    backgroundColor: colors.muted,
  },
  statusText: {
    color: colors.muted,
    fontSize: 12,
  },
  rule: {
    height: 1,
    marginTop: 15,
    marginBottom: 21,
    backgroundColor: colors.border,
  },
  cardTitle: {
    color: colors.text,
    fontSize: 20,
    fontWeight: "600",
    letterSpacing: -0.4,
  },
  cardBody: {
    marginTop: 8,
    color: colors.muted,
    fontSize: 14,
    lineHeight: 21,
  },
  disabledButton: {
    minHeight: 50,
    marginTop: 24,
    paddingHorizontal: 15,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.quiet,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
  },
  disabledButtonText: {
    color: colors.muted,
    fontSize: 14,
    fontWeight: "600",
  },
  buttonArrow: {
    color: colors.muted,
    fontSize: 18,
  },
  note: {
    marginTop: 10,
    color: colors.muted,
    fontSize: 11,
    textAlign: "center",
  },
  footer: {
    marginTop: "auto",
    paddingTop: 32,
  },
  footerText: {
    color: colors.muted,
    fontSize: 11,
    lineHeight: 17,
  },
  footerVersion: {
    marginTop: 8,
    color: "#737D6D",
    fontSize: 10,
  },
});
